#!/usr/bin/env bash
# Integration tests for Deckhouse MCP server running in Kind.
#
# Usage: bash tests/integration/test.sh
#
# Requires: curl, jq, kubectl (port-forward to MCP server on localhost:8080).
set -euo pipefail

MCP_BASE_URL="${MCP_BASE_URL:-http://localhost:8080}"
KUBE_CONTEXT="${KUBE_CONTEXT:-kind-d8}"

# Counters.
PASSED=0
FAILED=0
SKIPPED=0
TOTAL=0

# Temp dir for SSE stream files.
TMPDIR=$(mktemp -d)
trap 'cleanup' EXIT

cleanup() {
  # Kill background curl processes.
  jobs -p 2>/dev/null | xargs kill 2>/dev/null || true
  rm -rf "$TMPDIR"
}

# --- SSE/MCP Helpers ----------------------------------------------------------

# Connect to SSE endpoint and return the session message endpoint URL.
# Writes SSE stream to $TMPDIR/sse_stream in background.
mcp_connect() {
  local stream_file="$TMPDIR/sse_stream"
  > "$stream_file"

  # Start background SSE listener.
  curl -sN "${MCP_BASE_URL}/sse" > "$stream_file" 2>/dev/null &
  local curl_pid=$!
  echo "$curl_pid" > "$TMPDIR/curl_pid"

  # Wait for the endpoint event.
  local endpoint=""
  for i in $(seq 1 30); do
    if [ -s "$stream_file" ]; then
      endpoint=$(grep '^data:' "$stream_file" | head -1 | sed 's/^data: *//')
      if [ -n "$endpoint" ]; then
        break
      fi
    fi
    sleep 0.5
  done

  if [ -z "$endpoint" ]; then
    echo "ERROR: Failed to get endpoint from SSE stream" >&2
    return 1
  fi

  # Resolve relative endpoint to absolute URL.
  if [[ "$endpoint" == /* ]]; then
    endpoint="${MCP_BASE_URL}${endpoint}"
  fi

  echo "$endpoint"
}

# Disconnect from SSE (kill background curl).
mcp_disconnect() {
  if [ -f "$TMPDIR/curl_pid" ]; then
    kill "$(cat "$TMPDIR/curl_pid")" 2>/dev/null || true
    rm -f "$TMPDIR/curl_pid"
  fi
}

# Send a JSON-RPC request and wait for the response on the SSE stream.
# Usage: mcp_request <endpoint> <method> <params_json> [id]
mcp_request() {
  local endpoint="$1"
  local method="$2"
  local params="${3:-null}"
  local id="${4:-1}"

  local stream_file="$TMPDIR/sse_stream"
  local line_count_before
  line_count_before=$(wc -l < "$stream_file" | tr -d ' ')

  # POST the JSON-RPC request.
  local body
  body=$(jq -n --arg method "$method" --argjson params "$params" --argjson id "$id" \
    '{"jsonrpc":"2.0","method":$method,"params":$params,"id":$id}')

  curl -sf -X POST "$endpoint" \
    -H "Content-Type: application/json" \
    -d "$body" >/dev/null 2>&1

  # Wait for response on SSE stream (new message event after our request).
  local response=""
  for i in $(seq 1 200); do
    # Extract all 'data:' lines after the initial endpoint event.
    local new_data
    new_data=$(tail -n +"$((line_count_before + 1))" "$stream_file" 2>/dev/null | grep '^data:' | sed 's/^data: *//' || true)

    if [ -n "$new_data" ]; then
      # Find a JSON-RPC response with our id.
      while IFS= read -r line; do
        if echo "$line" | jq -e --argjson id "$id" 'select(.id == $id)' >/dev/null 2>&1; then
          response="$line"
          break 2
        fi
      done <<< "$new_data"
    fi
    sleep 0.5
  done

  if [ -z "$response" ]; then
    echo '{"error":{"code":-1,"message":"Timeout waiting for SSE response"}}' >&2
    return 1
  fi

  echo "$response"
}

# Initialize MCP session. Returns the endpoint URL.
mcp_initialize() {
  local endpoint
  endpoint=$(mcp_connect) || return 1

  local resp
  resp=$(mcp_request "$endpoint" "initialize" '{
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {"name": "integration-test", "version": "1.0.0"}
  }' 0)

  # Check for error.
  if echo "$resp" | jq -e '.error' >/dev/null 2>&1; then
    echo "ERROR: Initialize failed: $(echo "$resp" | jq -r '.error.message')" >&2
    return 1
  fi

  # Send initialized notification (no id = notification).
  curl -sf -X POST "$endpoint" \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"notifications/initialized"}' >/dev/null 2>&1

  echo "$endpoint"
}

# Call an MCP tool. Returns the result JSON.
# Usage: mcp_call_tool <endpoint> <tool_name> [arguments_json] [request_id]
mcp_call_tool() {
  local endpoint="$1"
  local tool_name="$2"
  local arguments="${3:-}"
  if [ -z "$arguments" ]; then arguments='{}'; fi
  local id="${4:-$((RANDOM % 10000 + 100))}"

  local params
  params=$(jq -n --arg name "$tool_name" --argjson args "$arguments" \
    '{"name":$name,"arguments":$args}')

  local resp
  resp=$(mcp_request "$endpoint" "tools/call" "$params" "$id") || return 1

  # Check for JSON-RPC error.
  if echo "$resp" | jq -e '.error' >/dev/null 2>&1; then
    echo "TOOL ERROR: $(echo "$resp" | jq -r '.error.message')" >&2
    echo "$resp"
    return 1
  fi

  # Extract the text content from the result.
  echo "$resp" | jq -r '.result.content[0].text // .result'
}

# --- Assertions ---------------------------------------------------------------

assert_contains() {
  local text="$1"
  local pattern="$2"
  local msg="${3:-}"

  if echo "$text" | grep -q "$pattern"; then
    return 0
  else
    echo "  ASSERT FAILED: expected to contain '$pattern'${msg:+ ($msg)}" >&2
    echo "  Got: $(echo "$text" | head -5)" >&2
    return 1
  fi
}

assert_jq() {
  local json="$1"
  local expr="$2"
  local msg="${3:-}"

  if echo "$json" | jq -e "$expr" >/dev/null 2>&1; then
    return 0
  else
    echo "  ASSERT FAILED: jq '$expr' returned false${msg:+ ($msg)}" >&2
    echo "  Got: $(echo "$json" | head -5)" >&2
    return 1
  fi
}

# --- Test runner --------------------------------------------------------------

# Exit code 77 = skip (test not applicable in current environment).
run_test() {
  local name="$1"
  TOTAL=$((TOTAL + 1))
  echo -n "[$TOTAL] $name ... "

  local rc=0
  "$name" || rc=$?

  case $rc in
    0)  PASSED=$((PASSED + 1));   echo "PASS" ;;
    77) SKIPPED=$((SKIPPED + 1)); echo "SKIP (environment limitation)" ;;
    *)  FAILED=$((FAILED + 1));   echo "FAIL" ;;
  esac
}

# --- Cluster probes -----------------------------------------------------------

# Returns 0 when the deckhouse validating webhook endpoints are populated.
# The Deckhouse module-configs webhook depends on a healthy controller Pod
# inside d8-system; in long-lived dev clusters it can become unreachable
# while the rest of the API stays usable. Tests that mutate ModuleConfig
# objects must probe this before running.
deckhouse_webhook_reachable() {
  local endpoints
  endpoints=$(kubectl --context "$KUBE_CONTEXT" -n d8-system get endpoints deckhouse \
    -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null || echo "")
  [ -n "$endpoints" ]
}

# --- Test cases ---------------------------------------------------------------

test_get_cluster_status() {
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_GetClusterStatus") || return 1

  assert_jq "$result" '.nodes.total >= 1' "at least 1 node" || return 1
  assert_jq "$result" '.nodes.ready >= 1' "at least 1 ready node" || return 1
  assert_jq "$result" '.deckhouseVersion | length > 0' "deckhouse version present" || return 1
}

test_list_nodes() {
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_ListNodes") || return 1

  assert_jq "$result" '.nodes | length >= 1' "at least 1 node" || return 1
  assert_contains "$result" "control-plane" "Kind node name" || return 1
}

test_list_nodes_filter_ready() {
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_ListNodes" '{"status":"NODE_STATUS_FILTER_READY"}') || return 1

  assert_jq "$result" '.nodes | length >= 1' "at least 1 ready node" || return 1
  # All returned nodes should be Ready.
  assert_jq "$result" '[.nodes[].status] | all(. == "Ready")' "all nodes Ready" || return 1
}

test_list_node_groups() {
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_ListNodeGroups") || return 1

  assert_jq "$result" '.nodeGroups | length >= 1' "at least 1 node group" || return 1
}

test_list_static_instances() {
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_ListStaticInstances") || return 1

  # Empty list is OK for Kind — just verify no error.
  assert_jq "$result" '.instances | type == "array"' "instances is array" || return 1
}

test_list_unhealthy_pods() {
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_ListUnhealthyPods") || return 1

  # May be empty or have pods — just verify structure.
  assert_jq "$result" '.pods | type == "array"' "pods is array" || return 1
}

test_list_module_configs() {
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_ListModuleConfigs") || return 1

  assert_jq "$result" '.modules | length >= 5' "at least 5 modules" || return 1
  # Deckhouse itself should be in the list.
  assert_contains "$result" "deckhouse" "deckhouse module present" || return 1
}

test_list_module_configs_filter_enabled() {
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_ListModuleConfigs" '{"enabled":true}') || return 1

  assert_jq "$result" '.modules | length >= 1' "at least 1 enabled module" || return 1
}

test_list_deckhouse_releases() {
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_ListDeckhouseReleases") || return 1

  assert_jq "$result" '.releases | length >= 1' "at least 1 release" || return 1
}

test_create_ssh_credentials() {
  # Clean up any leftover from a previous run to ensure idempotent tests.
  kubectl --context "$KUBE_CONTEXT" delete sshcredentials integration-test-creds --ignore-not-found=true >/dev/null 2>&1 || true

  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_CreateSSHCredentials" '{
    "name": "integration-test-creds",
    "user": "testuser",
    "privateKey": "-----BEGIN OPENSSH PRIVATE KEY-----\ntest-key-data-for-integration\n-----END OPENSSH PRIVATE KEY-----"
  }') || return 1

  assert_jq "$result" '.name == "integration-test-creds"' "returned name matches" || return 1

  # Verify via kubectl.
  kubectl --context "$KUBE_CONTEXT" get sshcredentials integration-test-creds >/dev/null 2>&1 || {
    echo "  ASSERT FAILED: SSHCredentials not found via kubectl" >&2
    return 1
  }
}

test_create_static_instance() {
  # Clean up any leftover from a previous run.
  kubectl --context "$KUBE_CONTEXT" delete staticinstances integration-test-si --ignore-not-found=true >/dev/null 2>&1 || true

  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_CreateStaticInstance" '{
    "name": "integration-test-si",
    "address": "192.168.1.100",
    "credentialsRef": "integration-test-creds",
    "labels": {"node.deckhouse.io/group": "worker"}
  }') || return 1

  assert_jq "$result" '.name == "integration-test-si"' "returned name matches" || return 1

  # Verify via kubectl.
  kubectl --context "$KUBE_CONTEXT" get staticinstances integration-test-si >/dev/null 2>&1 || {
    echo "  ASSERT FAILED: StaticInstance not found via kubectl" >&2
    return 1
  }
}

test_add_worker_node() {
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_AddWorkerNode" '{
    "address": "192.168.1.200",
    "sshUser": "testuser",
    "privateKey": "-----BEGIN OPENSSH PRIVATE KEY-----\ntest-key-data-for-add-worker\n-----END OPENSSH PRIVATE KEY-----",
    "nodeGroup": "worker",
    "nodeName": "integration-test-worker",
    "timeoutSeconds": 5,
    "waitReady": true
  }') || {
    # AddWorkerNode may return an error or timeout — check the response.
    true
  }

  # Even with timeout, the response should contain the resource names.
  assert_contains "$result" "integration-test-worker" "node name in response" || return 1

  # Verify SSHCredentials and StaticInstance were created.
  kubectl --context "$KUBE_CONTEXT" get sshcredentials integration-test-worker-creds >/dev/null 2>&1 || {
    echo "  ASSERT FAILED: SSHCredentials for worker not found via kubectl" >&2
    return 1
  }
  kubectl --context "$KUBE_CONTEXT" get staticinstances integration-test-worker >/dev/null 2>&1 || {
    echo "  ASSERT FAILED: StaticInstance for worker not found via kubectl" >&2
    return 1
  }
}

# --- P1 read-only tests -------------------------------------------------------

test_get_node() {
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_GetNode" '{"name":"d8-control-plane"}') || return 1

  assert_jq "$result" '.node.name == "d8-control-plane"' "node name matches" || return 1
  assert_jq "$result" '.conditions | length >= 1' "conditions present" || return 1
  assert_jq "$result" '.capacity | has("cpu")' "capacity has cpu" || return 1
  assert_jq "$result" '.allocatable | has("memory")' "allocatable has memory" || return 1
}

test_get_node_not_found() {
  # Call with unknown node name; MCP tool error expected.
  local raw
  raw=$(mcp_call_tool "$ENDPOINT" "deckhouse_GetNode" '{"name":"nonexistent-node-xyz"}' 2>&1) || true
  assert_contains "$raw" "nonexistent-node-xyz" "error mentions the node name" || return 1
}

test_get_node_group() {
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_GetNodeGroup" '{"name":"master"}') || return 1

  assert_jq "$result" '.name == "master"' "node group name matches" || return 1
  assert_jq "$result" '.nodeNames | type == "array"' "nodeNames is array" || return 1
}

test_get_deckhouse_logs() {
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_GetDeckhouseLogs" '{"tail":20}') || return 1

  # The deckhouse controller pod may be scaled to 0 in a Kind demo cluster.
  # Accept either valid logs or the descriptive "pod not found" error.
  if echo "$result" | jq -e '.logs' >/dev/null 2>&1; then
    assert_jq "$result" '.logs | type == "string"' "logs is string" || return 1
  else
    assert_contains "$result" "deckhouse pod not found" "descriptive error when controller absent" || return 1
  fi
}

test_get_deckhouse_logs_grep() {
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_GetDeckhouseLogs" '{"tail":100,"grep":"time="}') || return 1

  if echo "$result" | jq -e '.logs' >/dev/null 2>&1; then
    assert_jq "$result" '.logs | type == "string"' "logs is string" || return 1
  else
    assert_contains "$result" "deckhouse pod not found" "descriptive error when controller absent" || return 1
  fi
}

test_get_module_config() {
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_GetModuleConfig" '{"name":"deckhouse"}') || return 1

  assert_jq "$result" '.name == "deckhouse"' "module name matches" || return 1
  assert_jq "$result" '.enabled == true' "deckhouse module is enabled" || return 1
}

test_get_deckhouse_release() {
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_GetDeckhouseRelease" '{"version":"v1.70.0"}') || return 1

  assert_jq "$result" '.version == "v1.70.0"' "release version matches" || return 1
  assert_jq "$result" '.phase | length > 0' "phase is set" || return 1
}

test_get_cluster_configuration() {
  # Requires the d8-cluster-configuration secret in kube-system. In CE Kind
  # clusters this secret is not always provisioned; skip when missing.
  if ! kubectl --context "$KUBE_CONTEXT" -n kube-system get secret d8-cluster-configuration >/dev/null 2>&1; then
    echo "SKIP: d8-cluster-configuration secret not provisioned in this cluster"
    return 77
  fi
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_GetClusterConfiguration") || return 1

  assert_contains "$result" "ClusterConfiguration" "configuration YAML has type" || return 1
  assert_contains "$result" "kubernetesVersion" "configuration has kubernetesVersion" || return 1
}

# --- P1 write tests -----------------------------------------------------------

test_create_node_group() {
  # Clean up any leftover from a previous run.
  kubectl --context "$KUBE_CONTEXT" delete nodegroups integration-test-ng --ignore-not-found=true >/dev/null 2>&1 || true

  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_CreateNodeGroup" '{
    "name": "integration-test-ng",
    "nodeType": "Static"
  }') || return 1

  assert_jq "$result" '.name == "integration-test-ng"' "node group name in response" || return 1

  kubectl --context "$KUBE_CONTEXT" get nodegroups integration-test-ng >/dev/null 2>&1 || {
    echo "  ASSERT FAILED: NodeGroup not found via kubectl" >&2
    return 1
  }
}

test_enable_module_idempotent() {
  # Requires the Deckhouse module-configs webhook to be reachable.
  if ! deckhouse_webhook_reachable; then
    echo "SKIP: deckhouse validating webhook unreachable"
    return 77
  fi
  local replicas
  replicas=$(kubectl --context "$KUBE_CONTEXT" -n d8-system get deployment deckhouse \
    -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
  if [ "${replicas:-0}" -eq 0 ]; then
    echo -n "(deckhouse at 0 replicas, webhook unavailable) "
    return 77  # SKIP
  fi

  # Enable an already-enabled module — tests idempotency (previousState: true, no actual change).
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_EnableModule" '{"name":"deckhouse"}') || return 1

  assert_jq "$result" '.success == true' "success is true" || return 1
  assert_jq "$result" '.previousState == true' "previousState is true (was already enabled)" || return 1
}

test_approve_release() {
  # ApproveRelease patches a DeckhouseRelease, which Deckhouse routes through
  # its deckhouse-releases validating webhook. Skip if the webhook is down.
  if ! deckhouse_webhook_reachable; then
    echo "SKIP: deckhouse validating webhook unreachable"
    return 77
  fi
  # v1.71.0 is in Pending phase — approve it (idempotent if re-run).
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_ApproveRelease" '{"version":"v1.71.0"}') || return 1

  assert_jq "$result" '.success == true' "success is true" || return 1

  # Verify annotation was set via kubectl.
  local approved
  approved=$(kubectl --context "$KUBE_CONTEXT" get deckhouserelease v1.71.0 \
    -o jsonpath='{.metadata.annotations.release\.deckhouse\.io/approved}' 2>/dev/null)
  if [ "$approved" != "true" ]; then
    echo "  ASSERT FAILED: release annotation not set (got: '$approved')" >&2
    return 1
  fi
}

test_wait_node_ready_timeout() {
  # integration-test-si (192.168.1.100) will never reach Running in Kind — expect timedOut=true.
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_WaitNodeReady" '{
    "name": "integration-test-si",
    "timeoutSeconds": 30,
    "intervalSeconds": 5
  }') || return 1

  assert_jq "$result" '.timedOut == true' "timedOut is true for fake node" || return 1
  assert_jq "$result" '.elapsed | length > 0' "elapsed is set" || return 1
}

test_delete_static_instance() {
  # Create a dedicated SI for deletion, then delete it via the MCP tool.
  kubectl --context "$KUBE_CONTEXT" apply -f - >/dev/null 2>&1 <<'EOF'
apiVersion: deckhouse.io/v1alpha2
kind: StaticInstance
metadata:
  name: integration-test-delete-si
spec:
  address: "192.168.1.250"
  credentialsRef:
    kind: SSHCredentials
    name: integration-test-creds
EOF

  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_DeleteStaticInstance" '{"name":"integration-test-delete-si"}') || return 1

  assert_jq "$result" '.success == true' "success is true" || return 1

  # Verify it's gone.
  if kubectl --context "$KUBE_CONTEXT" get staticinstances integration-test-delete-si >/dev/null 2>&1; then
    echo "  ASSERT FAILED: StaticInstance still exists after delete" >&2
    return 1
  fi
}

test_remove_node_no_static_instance() {
  # d8-control-plane has no StaticInstance in Kind — expect "not found" error.
  local raw
  raw=$(mcp_call_tool "$ENDPOINT" "deckhouse_RemoveNode" '{"name":"d8-control-plane"}' 2>&1) || true
  assert_contains "$raw" "not found" "error mentions not found" || return 1
}

# --- P1 write tests (gap) -----------------------------------------------------

test_disable_module_idempotent() {
  # Disable -> Enable round-trip on cert-manager. cert-manager is part of the
  # CE bundle and safe to toggle. Requires the Deckhouse module-configs
  # validating webhook to be reachable; if not, skip.
  if ! deckhouse_webhook_reachable; then
    echo "SKIP: deckhouse validating webhook unreachable"
    return 77
  fi

  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_DisableModule" '{"name":"cert-manager"}') || return 1
  assert_jq "$result" '.previousState | type == "boolean"' "previousState is boolean" || return 1

  # Re-enable to leave the cluster in a known state.
  mcp_call_tool "$ENDPOINT" "deckhouse_EnableModule" '{"name":"cert-manager"}' >/dev/null 2>&1 || true
}

# --- P2 read-only tests -------------------------------------------------------

test_get_node_events() {
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_GetNodeEvents" '{"name":"d8-control-plane"}') || return 1

  assert_jq "$result" '.events | type == "array"' "events is array" || return 1
}

test_get_node_events_not_found() {
  # Unknown node — handler returns empty list (no error).
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_GetNodeEvents" '{"name":"nonexistent-node-xyz"}') || return 1

  assert_jq "$result" '.events | length == 0' "no events for unknown node" || return 1
}

test_get_pod_logs() {
  # Use the deckhouse pod itself in d8-system. Discover the actual pod name.
  # The deckhouse pod has multiple containers — pick the one named "deckhouse".
  local pod_name
  pod_name=$(kubectl --context "$KUBE_CONTEXT" -n d8-system get pods \
    -l app=deckhouse -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
  if [ -z "$pod_name" ]; then
    echo "SKIP: no deckhouse pod found"
    return 77
  fi

  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_GetPodLogs" \
    "$(jq -n --arg ns "d8-system" --arg p "$pod_name" '{namespace:$ns, pod:$p, container:"deckhouse", tail:10}')") || return 1

  assert_jq "$result" '.logs | type == "string"' "logs is string" || return 1
}

test_get_pod_logs_not_found() {
  local raw
  raw=$(mcp_call_tool "$ENDPOINT" "deckhouse_GetPodLogs" \
    '{"namespace":"d8-system","pod":"nonexistent-pod-xyz","tail":10}' 2>&1) || true
  assert_contains "$raw" "nonexistent-pod-xyz" "error mentions the pod name" || return 1
}

test_get_static_instance() {
  # Create a SI to read back, then clean up.
  kubectl --context "$KUBE_CONTEXT" delete staticinstances integration-test-get-si --ignore-not-found=true >/dev/null 2>&1 || true

  # Need credentials first.
  kubectl --context "$KUBE_CONTEXT" delete sshcredentials integration-test-get-si-creds --ignore-not-found=true >/dev/null 2>&1 || true
  mcp_call_tool "$ENDPOINT" "deckhouse_CreateSSHCredentials" '{
    "name": "integration-test-get-si-creds",
    "user": "testuser",
    "privateKey": "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----"
  }' >/dev/null 2>&1 || true

  mcp_call_tool "$ENDPOINT" "deckhouse_CreateStaticInstance" '{
    "name": "integration-test-get-si",
    "address": "192.168.1.150",
    "credentialsRef": "integration-test-get-si-creds"
  }' >/dev/null 2>&1 || true

  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_GetStaticInstance" '{"name":"integration-test-get-si"}') || {
    kubectl --context "$KUBE_CONTEXT" delete staticinstances integration-test-get-si --ignore-not-found=true >/dev/null 2>&1 || true
    kubectl --context "$KUBE_CONTEXT" delete sshcredentials integration-test-get-si-creds --ignore-not-found=true >/dev/null 2>&1 || true
    return 1
  }

  assert_jq "$result" '.name == "integration-test-get-si"' "name matches" || return 1
  assert_jq "$result" '.address == "192.168.1.150"' "address matches" || return 1

  # Cleanup-after.
  kubectl --context "$KUBE_CONTEXT" delete staticinstances integration-test-get-si --ignore-not-found=true >/dev/null 2>&1 || true
  kubectl --context "$KUBE_CONTEXT" delete sshcredentials integration-test-get-si-creds --ignore-not-found=true >/dev/null 2>&1 || true
}

test_get_static_instance_not_found() {
  local raw
  raw=$(mcp_call_tool "$ENDPOINT" "deckhouse_GetStaticInstance" '{"name":"nonexistent-si-xyz"}' 2>&1) || true
  assert_contains "$raw" "not found" "error mentions not found" || return 1
}

test_list_modules() {
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_ListModules") || return 1

  # In a real Deckhouse CE cluster there is at least one Module.
  assert_jq "$result" '.modules | type == "array"' "modules is array" || return 1
}

test_get_static_cluster_configuration() {
  # The static-cluster-configuration.yaml key is optional. Accept either
  # success with valid YAML or a descriptive "not found" error.
  local raw
  raw=$(mcp_call_tool "$ENDPOINT" "deckhouse_GetStaticClusterConfiguration" 2>&1) || true

  if echo "$raw" | jq -e '.yaml' >/dev/null 2>&1; then
    assert_contains "$raw" "StaticClusterConfiguration" "configuration has type" || return 1
  else
    # No StaticClusterConfiguration in this cluster — accept the not-found error.
    assert_contains "$raw" "not found" "error mentions not found" || return 1
  fi
}

# --- P2 write tests -----------------------------------------------------------

test_update_module_settings_not_found() {
  local raw
  raw=$(mcp_call_tool "$ENDPOINT" "deckhouse_UpdateModuleSettings" '{
    "name": "nonexistent-module-xyz",
    "settings": {"key":"value"}
  }' 2>&1) || true
  assert_contains "$raw" "nonexistent-module-xyz" "error mentions module name" || return 1
}

# CordonNode + UncordonNode share state — run as a coupled pair: cordon →
# verify → uncordon → verify, so the cluster is left in its original state.
test_cordon_node() {
  local node_name="d8-control-plane"
  # Make sure node is uncordoned before the test.
  kubectl --context "$KUBE_CONTEXT" uncordon "$node_name" >/dev/null 2>&1 || true

  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_CordonNode" "$(jq -n --arg n "$node_name" '{name:$n}')") || return 1

  # CordonNodeResponse exposes previousState (was the node already cordoned).
  assert_jq "$result" '.previousState == false' "node was not cordoned before the call" || return 1

  # Verify with kubectl that the spec.unschedulable is set.
  local unschedulable
  unschedulable=$(kubectl --context "$KUBE_CONTEXT" get node "$node_name" \
    -o jsonpath='{.spec.unschedulable}' 2>/dev/null || echo "")
  if [ "$unschedulable" != "true" ]; then
    echo "expected node spec.unschedulable=true, got: '$unschedulable'"
    return 1
  fi
}

test_uncordon_node() {
  local node_name="d8-control-plane"
  # Pre-cordon so uncordon has work to do.
  kubectl --context "$KUBE_CONTEXT" cordon "$node_name" >/dev/null 2>&1 || true

  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_UncordonNode" "$(jq -n --arg n "$node_name" '{name:$n}')") || return 1

  # UncordonNodeResponse exposes previousState (was the node cordoned before).
  assert_jq "$result" '.previousState == true' "node was cordoned before the call" || return 1

  # Verify with kubectl that spec.unschedulable is gone.
  local unschedulable
  unschedulable=$(kubectl --context "$KUBE_CONTEXT" get node "$node_name" \
    -o jsonpath='{.spec.unschedulable}' 2>/dev/null || echo "")
  if [ "$unschedulable" = "true" ]; then
    echo "expected node spec.unschedulable to be unset/false, got: '$unschedulable'"
    return 1
  fi
}

test_drain_node_single_node_protected() {
  # Single-node Kind: draining the only control-plane is destructive.
  # We only verify the DrainNode tool is registered and rejects with a
  # short timeout (so the cluster remains operational). The handler will
  # cordon the node, attempt eviction (DaemonSets are skipped), and likely
  # time out. Both timed_out=true and a normal completion are acceptable —
  # what matters is that the tool returns a structured response.
  local node_count
  node_count=$(kubectl --context "$KUBE_CONTEXT" get nodes --no-headers 2>/dev/null | wc -l | tr -d ' ')
  if [ "$node_count" -lt 2 ]; then
    echo "SKIP: drain on single-node Kind would destabilise the cluster"
    # Restore cordon state in case prior cordon test left node cordoned.
    kubectl --context "$KUBE_CONTEXT" uncordon d8-control-plane >/dev/null 2>&1 || true
    return 77
  fi

  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_DrainNode" '{
    "name": "d8-control-plane",
    "timeoutSeconds": 30
  }') || return 1

  assert_jq "$result" '.cordoned | type == "boolean"' "cordoned is boolean" || return 1
  # Restore state.
  kubectl --context "$KUBE_CONTEXT" uncordon d8-control-plane >/dev/null 2>&1 || true
}

test_delete_ssh_credentials() {
  # Pre-create via kubectl (avoids dependency on CreateSSHCredentials test order).
  local name="integration-test-delete-creds"
  kubectl --context "$KUBE_CONTEXT" delete sshcredentials "$name" --ignore-not-found=true >/dev/null 2>&1 || true
  cat <<EOF | kubectl --context "$KUBE_CONTEXT" apply -f - >/dev/null 2>&1
apiVersion: deckhouse.io/v1alpha2
kind: SSHCredentials
metadata:
  name: ${name}
spec:
  user: testuser
  privateSSHKey: dGVzdA==
EOF

  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_DeleteSSHCredentials" "$(jq -n --arg n "$name" '{name:$n}')") || {
    kubectl --context "$KUBE_CONTEXT" delete sshcredentials "$name" --ignore-not-found=true >/dev/null 2>&1 || true
    return 1
  }

  assert_jq "$result" '.deleted == true' "deleted is true" || return 1

  # Verify gone.
  if kubectl --context "$KUBE_CONTEXT" get sshcredentials "$name" >/dev/null 2>&1; then
    echo "SSHCredentials still present after delete"
    return 1
  fi
}

test_delete_ssh_credentials_not_found() {
  local raw
  raw=$(mcp_call_tool "$ENDPOINT" "deckhouse_DeleteSSHCredentials" \
    '{"name":"nonexistent-creds-xyz"}' 2>&1) || true
  assert_contains "$raw" "not found" "error mentions not found" || return 1
}

test_delete_node_group() {
  local name="integration-test-delete-ng"
  kubectl --context "$KUBE_CONTEXT" delete nodegroups "$name" --ignore-not-found=true >/dev/null 2>&1 || true
  cat <<EOF | kubectl --context "$KUBE_CONTEXT" apply -f - >/dev/null 2>&1
apiVersion: deckhouse.io/v1
kind: NodeGroup
metadata:
  name: ${name}
spec:
  nodeType: Static
EOF

  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_DeleteNodeGroup" "$(jq -n --arg n "$name" '{name:$n}')") || {
    kubectl --context "$KUBE_CONTEXT" delete nodegroups "$name" --ignore-not-found=true >/dev/null 2>&1 || true
    return 1
  }

  assert_jq "$result" '.deleted == true' "deleted is true" || return 1
}

test_delete_node_group_not_found() {
  local raw
  raw=$(mcp_call_tool "$ENDPOINT" "deckhouse_DeleteNodeGroup" \
    '{"name":"nonexistent-ng-xyz"}' 2>&1) || true
  assert_contains "$raw" "not found" "error mentions not found" || return 1
}

test_update_kubernetes_version_invalid_format() {
  # Reject "abc" — not a MAJOR.MINOR string. Either the handler or the
  # Deckhouse webhook should refuse it.
  local raw
  raw=$(mcp_call_tool "$ENDPOINT" "deckhouse_UpdateKubernetesVersion" \
    '{"version":"abc"}' 2>&1) || true
  # We only assert there is *some* error message; the exact wording differs
  # between handler-side validation and Deckhouse hook rejection.
  if echo "$raw" | jq -e '.updated' >/dev/null 2>&1; then
    echo "expected error, got success"
    return 1
  fi
}

test_list_module_sources() {
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_ListModuleSources") || return 1
  assert_jq "$result" '.sources | type == "array"' "sources is array" || return 1
}

test_create_module_source() {
  local name="integration-test-source"
  kubectl --context "$KUBE_CONTEXT" delete modulesources "$name" --ignore-not-found=true >/dev/null 2>&1 || true

  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_CreateModuleSource" "$(jq -n --arg n "$name" '{name:$n, registry:"registry.deckhouse.io/deckhouse/ce/modules"}')") || {
    kubectl --context "$KUBE_CONTEXT" delete modulesources "$name" --ignore-not-found=true >/dev/null 2>&1 || true
    return 1
  }

  assert_jq "$result" '.created == true' "created is true" || return 1
  assert_jq "$result" '.name == "integration-test-source"' "name echoed" || return 1

  kubectl --context "$KUBE_CONTEXT" delete modulesources "$name" --ignore-not-found=true >/dev/null 2>&1 || true
}

test_create_module_source_already_exists() {
  local name="integration-test-source-dup"
  # Pre-create via kubectl to be deterministic.
  cat <<EOF | kubectl --context "$KUBE_CONTEXT" apply -f - >/dev/null 2>&1
apiVersion: deckhouse.io/v1alpha1
kind: ModuleSource
metadata:
  name: ${name}
spec:
  registry:
    repo: registry.deckhouse.io/deckhouse/ce/modules
EOF

  local raw
  raw=$(mcp_call_tool "$ENDPOINT" "deckhouse_CreateModuleSource" "$(jq -n --arg n "$name" '{name:$n, registry:"registry.deckhouse.io/deckhouse/ce/modules"}')" 2>&1) || true

  kubectl --context "$KUBE_CONTEXT" delete modulesources "$name" --ignore-not-found=true >/dev/null 2>&1 || true

  assert_contains "$raw" "already" "error mentions already" || return 1
}

test_list_module_update_policies() {
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_ListModuleUpdatePolicies") || return 1
  assert_jq "$result" '.policies | type == "array"' "policies is array" || return 1
}

test_create_module_update_policy() {
  # The Deckhouse update-policies validating webhook intercepts BOTH create
  # and delete on moduleupdatepolicies. Skip when it is unreachable: not only
  # would the handler call fail, but cleanup of leftover resources from past
  # runs would also be blocked.
  if ! deckhouse_webhook_reachable; then
    echo "SKIP: deckhouse update-policies webhook unreachable"
    return 77
  fi
  local name="integration-test-policy"
  kubectl --context "$KUBE_CONTEXT" delete moduleupdatepolicies "$name" --ignore-not-found=true >/dev/null 2>&1 || true

  local params
  params=$(jq -n --arg n "$name" '{
    name: $n,
    updateMode: "Manual",
    matchLabels: {"module": "integration-test"}
  }')

  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_CreateModuleUpdatePolicy" "$params") || {
    kubectl --context "$KUBE_CONTEXT" delete moduleupdatepolicies "$name" --ignore-not-found=true >/dev/null 2>&1 || true
    return 1
  }

  assert_jq "$result" '.created == true' "created is true" || return 1
  assert_jq "$result" '.name == "integration-test-policy"' "name echoed" || return 1

  # Verify the matchLabels round-trip via kubectl.
  local got_label
  got_label=$(kubectl --context "$KUBE_CONTEXT" get moduleupdatepolicy "$name" \
    -o jsonpath='{.spec.moduleReleaseSelector.labelSelector.matchLabels.module}' 2>/dev/null || echo "")
  if [ "$got_label" != "integration-test" ]; then
    echo "  ASSERT FAILED: spec.moduleReleaseSelector.labelSelector.matchLabels.module = '$got_label', expected 'integration-test'" >&2
    kubectl --context "$KUBE_CONTEXT" delete moduleupdatepolicies "$name" --ignore-not-found=true >/dev/null 2>&1 || true
    return 1
  fi

  kubectl --context "$KUBE_CONTEXT" delete moduleupdatepolicies "$name" --ignore-not-found=true >/dev/null 2>&1 || true
}

test_create_module_update_policy_missing_match_labels() {
  # Without matchLabels the handler short-circuits with errMatchLabelsRequired
  # before reaching the API server.
  local raw
  raw=$(mcp_call_tool "$ENDPOINT" "deckhouse_CreateModuleUpdatePolicy" \
    '{"name":"integration-test-no-selector","updateMode":"Auto"}' 2>&1) || true

  assert_contains "$raw" "match_labels is required" "error mentions match_labels is required" || return 1
}

test_create_module_update_policy_already_exists() {
  # Same webhook dependency as test_create_module_update_policy.
  if ! deckhouse_webhook_reachable; then
    echo "SKIP: deckhouse update-policies webhook unreachable"
    return 77
  fi
  local name="integration-test-policy-dup"
  cat <<EOF | kubectl --context "$KUBE_CONTEXT" apply -f - >/dev/null 2>&1
apiVersion: deckhouse.io/v1alpha1
kind: ModuleUpdatePolicy
metadata:
  name: ${name}
spec:
  update:
    mode: Manual
  moduleReleaseSelector:
    labelSelector:
      matchLabels:
        module: integration-test
EOF

  local params
  params=$(jq -n --arg n "$name" '{
    name: $n,
    updateMode: "Manual",
    matchLabels: {"module": "integration-test"}
  }')

  local raw
  raw=$(mcp_call_tool "$ENDPOINT" "deckhouse_CreateModuleUpdatePolicy" "$params" 2>&1) || true

  kubectl --context "$KUBE_CONTEXT" delete moduleupdatepolicies "$name" --ignore-not-found=true >/dev/null 2>&1 || true

  assert_contains "$raw" "already" "error mentions already" || return 1
}

# --- P3 tests -----------------------------------------------------------------

test_set_module_maintenance_enable_disable() {
  # Enter maintenance, exit maintenance — verifies both branches and the
  # round-trip leaves the cluster in its original state. Requires the
  # Deckhouse module-configs webhook to be reachable.
  if ! deckhouse_webhook_reachable; then
    echo "SKIP: deckhouse validating webhook unreachable"
    return 77
  fi

  local module="cert-manager"

  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_SetModuleMaintenance" "$(jq -n --arg n "$module" '{name:$n, enabled:true}')") || return 1
  assert_jq "$result" '.maintenanceEnabled == true' "maintenance enabled" || return 1

  # Verify spec.maintenance is set on the cluster.
  local field
  field=$(kubectl --context "$KUBE_CONTEXT" get moduleconfig "$module" \
    -o jsonpath='{.spec.maintenance}' 2>/dev/null || echo "")
  if [ "$field" != "NoResourceReconciliation" ]; then
    echo "expected spec.maintenance=NoResourceReconciliation, got: '$field'"
    return 1
  fi

  # Disable: spec.maintenance should be removed.
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_SetModuleMaintenance" "$(jq -n --arg n "$module" '{name:$n, enabled:false}')") || return 1
  assert_jq "$result" '.maintenanceEnabled == false' "maintenance disabled" || return 1

  field=$(kubectl --context "$KUBE_CONTEXT" get moduleconfig "$module" \
    -o jsonpath='{.spec.maintenance}' 2>/dev/null || echo "")
  if [ -n "$field" ]; then
    echo "expected spec.maintenance to be unset, got: '$field'"
    return 1
  fi
}

test_set_module_maintenance_idempotent() {
  # Two consecutive enable calls should both succeed (server-side merge patch is idempotent).
  if ! deckhouse_webhook_reachable; then
    echo "SKIP: deckhouse validating webhook unreachable"
    return 77
  fi

  local module="cert-manager"

  mcp_call_tool "$ENDPOINT" "deckhouse_SetModuleMaintenance" "$(jq -n --arg n "$module" '{name:$n, enabled:true}')" >/dev/null || return 1
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_SetModuleMaintenance" "$(jq -n --arg n "$module" '{name:$n, enabled:true}')") || return 1
  assert_jq "$result" '.maintenanceEnabled == true' "second call still reports enabled=true" || return 1

  # Cleanup: leave maintenance off.
  mcp_call_tool "$ENDPOINT" "deckhouse_SetModuleMaintenance" "$(jq -n --arg n "$module" '{name:$n, enabled:false}')" >/dev/null || true
}

test_create_node_group_configuration() {
  if ! kubectl --context "$KUBE_CONTEXT" get crd nodegroupconfigurations.deckhouse.io >/dev/null 2>&1; then
    echo "SKIP: NodeGroupConfiguration CRD not installed in this cluster"
    return 77
  fi

  local name="integration-test-ngc"
  kubectl --context "$KUBE_CONTEXT" delete nodegroupconfigurations "$name" --ignore-not-found=true >/dev/null 2>&1 || true

  local params
  params=$(jq -n --arg n "$name" '{
    name: $n,
    content: "#!/bin/bash\necho ok",
    nodeGroups: ["worker"],
    weight: 200
  }')

  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_CreateNodeGroupConfiguration" "$params") || {
    kubectl --context "$KUBE_CONTEXT" delete nodegroupconfigurations "$name" --ignore-not-found=true >/dev/null 2>&1 || true
    return 1
  }

  assert_jq "$result" '.created == true' "created is true" || return 1
  assert_jq "$result" '.name == "integration-test-ngc"' "name echoed" || return 1

  kubectl --context "$KUBE_CONTEXT" delete nodegroupconfigurations "$name" --ignore-not-found=true >/dev/null 2>&1 || true
}

test_create_node_group_configuration_already_exists() {
  if ! kubectl --context "$KUBE_CONTEXT" get crd nodegroupconfigurations.deckhouse.io >/dev/null 2>&1; then
    echo "SKIP: NodeGroupConfiguration CRD not installed in this cluster"
    return 77
  fi

  local name="integration-test-ngc-dup"
  cat <<EOF | kubectl --context "$KUBE_CONTEXT" apply -f - >/dev/null 2>&1
apiVersion: deckhouse.io/v1alpha1
kind: NodeGroupConfiguration
metadata:
  name: ${name}
spec:
  content: "#!/bin/bash\necho hello"
  nodeGroups: ["worker"]
  weight: 100
EOF

  local raw
  raw=$(mcp_call_tool "$ENDPOINT" "deckhouse_CreateNodeGroupConfiguration" \
    "$(jq -n --arg n "$name" '{name:$n, content:"#!/bin/bash\necho ok", nodeGroups:["worker"]}')" 2>&1) || true

  kubectl --context "$KUBE_CONTEXT" delete nodegroupconfigurations "$name" --ignore-not-found=true >/dev/null 2>&1 || true

  assert_contains "$raw" "already" "error mentions already" || return 1
}

test_delete_module_source() {
  # Create a module source without active releases, then delete via MCP.
  local name="integration-test-source-del"
  cat <<EOF | kubectl --context "$KUBE_CONTEXT" apply -f - >/dev/null 2>&1
apiVersion: deckhouse.io/v1alpha1
kind: ModuleSource
metadata:
  name: ${name}
spec:
  registry:
    repo: registry.deckhouse.io/deckhouse/ce/modules
EOF

  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_DeleteModuleSource" "$(jq -n --arg n "$name" '{name:$n}')") || {
    kubectl --context "$KUBE_CONTEXT" delete modulesources "$name" --ignore-not-found=true >/dev/null 2>&1 || true
    return 1
  }

  assert_jq "$result" '.deleted == true' "deleted is true" || return 1

  # Defensive cleanup if the handler somehow returned success without deleting.
  kubectl --context "$KUBE_CONTEXT" delete modulesources "$name" --ignore-not-found=true >/dev/null 2>&1 || true
}

test_delete_module_source_not_found() {
  local raw
  raw=$(mcp_call_tool "$ENDPOINT" "deckhouse_DeleteModuleSource" \
    '{"name":"nonexistent-source-xyz"}' 2>&1) || true
  assert_contains "$raw" "not found" "error mentions not found" || return 1
}

test_list_module_releases() {
  # Use "deckhouse" as the module name — DeckhouseRelease itself is the
  # canonical example. In a real CE cluster there will likely be at least
  # one ModuleRelease for an installed module; if not, an empty array is
  # also acceptable.
  local result
  result=$(mcp_call_tool "$ENDPOINT" "deckhouse_ListModuleReleases" '{"moduleName":"deckhouse"}') || return 1
  assert_jq "$result" '.releases | type == "array"' "releases is array" || return 1
}

test_list_module_releases_empty_module_name() {
  # Required field — handler must reject empty moduleName.
  local raw
  raw=$(mcp_call_tool "$ENDPOINT" "deckhouse_ListModuleReleases" '{"moduleName":""}' 2>&1) || true
  if echo "$raw" | jq -e '.releases' >/dev/null 2>&1; then
    echo "expected validation error, got success"
    return 1
  fi
  # Either handler error message OR a JSON-RPC validation error is OK.
  assert_contains "$raw" "module" "error mentions module" || return 1
}

# --- Main ---------------------------------------------------------------------

main() {
  echo "========================================"
  echo "Deckhouse MCP Integration Tests"
  echo "========================================"
  echo "MCP server: $MCP_BASE_URL"
  echo "Kube context: $KUBE_CONTEXT"
  echo ""

  # Initialize MCP session.
  echo "Connecting to MCP server..."
  ENDPOINT=$(mcp_initialize) || {
    echo "FATAL: Failed to initialize MCP session."
    exit 1
  }
  echo "Connected. Session endpoint: $ENDPOINT"
  echo ""

  # Run all tests.
  # Read-only tests first.
  run_test test_get_cluster_status
  run_test test_list_nodes
  run_test test_list_nodes_filter_ready
  run_test test_list_node_groups
  run_test test_list_static_instances
  run_test test_list_unhealthy_pods
  run_test test_list_module_configs
  run_test test_list_module_configs_filter_enabled
  run_test test_list_deckhouse_releases

  # Write tests (create resources).
  run_test test_create_ssh_credentials
  run_test test_create_static_instance

  # Composite write test (takes ~60s due to timeout).
  echo ""
  echo "Note: AddWorkerNode test will wait ~60s for timeout..."
  run_test test_add_worker_node

  # --- P1 read-only tests ---
  echo ""
  echo "--- P1 read-only tests ---"
  run_test test_get_node
  run_test test_get_node_not_found
  run_test test_get_node_group
  run_test test_get_deckhouse_logs
  run_test test_get_deckhouse_logs_grep
  run_test test_get_module_config
  run_test test_get_deckhouse_release
  run_test test_get_cluster_configuration

  # --- P1 write tests ---
  echo ""
  echo "--- P1 write tests ---"
  run_test test_create_node_group
  run_test test_enable_module_idempotent
  run_test test_approve_release

  echo ""
  echo "Note: WaitNodeReady test will wait ~5s for timeout..."
  run_test test_wait_node_ready_timeout

  run_test test_delete_static_instance
  run_test test_remove_node_no_static_instance

  # P1 gap (added with P3 work).
  run_test test_disable_module_idempotent

  # --- P2 read-only tests ---
  echo ""
  echo "--- P2 read-only tests ---"
  run_test test_get_node_events
  run_test test_get_node_events_not_found
  run_test test_get_pod_logs
  run_test test_get_pod_logs_not_found
  run_test test_get_static_instance
  run_test test_get_static_instance_not_found
  run_test test_list_modules
  run_test test_get_static_cluster_configuration

  # --- P2 write tests ---
  echo ""
  echo "--- P2 write tests ---"
  run_test test_update_module_settings_not_found
  run_test test_cordon_node
  run_test test_uncordon_node
  echo ""
  echo "Note: DrainNode test will run with timeout=30s (or skip on single-node Kind)..."
  run_test test_drain_node_single_node_protected
  run_test test_delete_ssh_credentials
  run_test test_delete_ssh_credentials_not_found
  run_test test_delete_node_group
  run_test test_delete_node_group_not_found
  run_test test_update_kubernetes_version_invalid_format
  run_test test_list_module_sources
  run_test test_create_module_source
  run_test test_create_module_source_already_exists
  run_test test_list_module_update_policies
  run_test test_create_module_update_policy
  run_test test_create_module_update_policy_missing_match_labels
  run_test test_create_module_update_policy_already_exists

  # --- P3 tests ---
  echo ""
  echo "--- P3 tests ---"
  run_test test_set_module_maintenance_enable_disable
  run_test test_set_module_maintenance_idempotent
  run_test test_create_node_group_configuration
  run_test test_create_node_group_configuration_already_exists
  run_test test_delete_module_source
  run_test test_delete_module_source_not_found
  run_test test_list_module_releases
  run_test test_list_module_releases_empty_module_name

  # Disconnect SSE.
  mcp_disconnect

  # Summary.
  echo ""
  echo "========================================"
  echo "Results: $PASSED passed, $FAILED failed, $SKIPPED skipped, $TOTAL total"
  echo "========================================"

  if [ "$FAILED" -gt 0 ]; then
    exit 1
  fi
}

main "$@"
