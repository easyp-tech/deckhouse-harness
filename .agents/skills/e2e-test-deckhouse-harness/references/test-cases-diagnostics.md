# Test Cases — DiagnosticsAPI (11 tools)

All tools in this group are **read-only**. No cleanup required.

Fixtures available:
- NodeGroups: `master` (CloudEphemeral, ready:1, nodes:1), `worker` (Static, ready:0, nodes:0)
- ModuleConfigs: `deckhouse`, `global`, `prometheus`, `ingress-nginx`, `cert-manager` (enabled), `node-manager` (disabled)
- DeckhouseReleases: `v1.70.0` (Deployed), `v1.71.0` (Pending)
- Kind node: `d8-control-plane` (the actual Kubernetes node)

---

## 1. deckhouse_GetClusterStatus

**Call:**
```bash
bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_GetClusterStatus '{}'
```

**Assertions:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh deckhouse_GetClusterStatus '{}')
DATA=$(echo "$RESP" | jq -r '.result.content[0].text | fromjson')

# No error
echo "$RESP" | jq -e '.result.isError // false | not'

# node_groups contains master and worker
echo "$DATA" | jq -e '[.nodeGroups[].name] | contains(["master","worker"])'

# deckhouse_version is from the Deployed release
echo "$DATA" | jq -e '.deckhouseVersion == "v1.70.0"'

# pending_releases contains v1.71.0
echo "$DATA" | jq -e '[.pendingReleases[].version] | contains(["v1.71.0"])'

# nodes.total >= 1 (Kind control-plane node is present)
echo "$DATA" | jq -e '.nodes.total >= 1'
```

**PASS:** All assertions exit 0.
**FAIL:** Any assertion fails or `isError: true`.

---

## 2. deckhouse_ListNodes

**Call (no filter):**
```bash
bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListNodes '{}'
```

**Assertions:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh deckhouse_ListNodes '{}')
DATA=$(echo "$RESP" | jq -r '.result.content[0].text | fromjson')
echo "$RESP" | jq -e '.result.isError // false | not'
echo "$DATA" | jq -e '.nodes | length >= 1'
# Kind node name contains "d8"
echo "$DATA" | jq -e '[.nodes[].name] | map(contains("d8")) | any'
```

**Call (filter by status=READY):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListNodes '{"status":"NODE_STATUS_FILTER_READY"}')
echo "$RESP" | jq -e '.result.isError // false | not'
# All returned nodes have status Ready
echo "$RESP" | jq -r '.result.content[0].text | fromjson | .nodes[].status' | grep -v "^Ready$" | wc -l | grep -q '^0$'
```

**PASS:** All assertions exit 0.
**FAIL:** Any assertion fails or `isError: true`.

---

## 3. deckhouse_ListNodeGroups

**Call:**
```bash
bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListNodeGroups '{}'
```

**Assertions:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh deckhouse_ListNodeGroups '{}')
DATA=$(echo "$RESP" | jq -r '.result.content[0].text | fromjson')
echo "$RESP" | jq -e '.result.isError // false | not'
echo "$DATA" | jq -e '.nodeGroups | length == 2'
echo "$DATA" | jq -e '[.nodeGroups[].name] | contains(["master","worker"])'
# master: nodeType=CloudEphemeral, ready=1
echo "$DATA" | jq -e '.nodeGroups[] | select(.name=="master") | .nodeType == "CloudEphemeral" and .ready == 1'
# worker: nodeType=Static
echo "$DATA" | jq -e '.nodeGroups[] | select(.name=="worker") | .nodeType == "Static"'
```

**PASS:** All assertions exit 0.
**FAIL:** Any assertion fails or `isError: true`.

---

## 4. deckhouse_ListStaticInstances

**Call (no filter):**
```bash
bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListStaticInstances '{}'
```

**Assertions:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh deckhouse_ListStaticInstances '{}')
echo "$RESP" | jq -e '.result.isError // false | not'
# No static instances seeded by fixtures — empty list is valid
echo "$RESP" | jq -r '.result.content[0].text | fromjson | .instances | length >= 0'
```

**Call (with phase filter):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListStaticInstances '{"phase":"STATIC_INSTANCE_PHASE_RUNNING"}')
echo "$RESP" | jq -e '.result.isError // false | not'
```

**PASS:** No error; `instances` field is an array (possibly empty).
**FAIL:** `isError: true`.

---

## 5. deckhouse_ListUnhealthyPods

**Call (all namespaces):**
```bash
bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListUnhealthyPods '{}'
```

**Assertions:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh deckhouse_ListUnhealthyPods '{}')
echo "$RESP" | jq -e '.result.isError // false | not'
# pods field must be an array (empty or non-empty)
echo "$RESP" | jq -r '.result.content[0].text | fromjson | .pods | type == "array" or .pods == null'
```

**Call (filter by namespace):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListUnhealthyPods '{"namespace":"kube-system"}')
echo "$RESP" | jq -e '.result.isError // false | not'
```

**PASS:** Both calls return without error; `pods` is an array.
**FAIL:** `isError: true`.

---

## 6. deckhouse_GetNode

**Note:** Requires the actual Kind node name. Get it first:
```bash
NODE_NAME=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListNodes '{}' \
  | jq -r '.result.content[0].text | fromjson | .nodes[0].name')
echo "Node name: $NODE_NAME"
```

**Call:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_GetNode "{\"name\":\"$NODE_NAME\"}")
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
DATA=$(echo "$RESP" | jq -r '.result.content[0].text | fromjson')
echo "$DATA" | jq -e ".node.name == \"$NODE_NAME\""
# conditions array is present
echo "$DATA" | jq -e '.conditions | type == "array"'
# allocatable and capacity maps are present
echo "$DATA" | jq -e '.allocatable | type == "object"'
echo "$DATA" | jq -e '.capacity | type == "object"'
# cpu and memory exist
echo "$DATA" | jq -e '.allocatable | has("cpu") and has("memory")'
```

**PASS:** All assertions exit 0.
**FAIL:** Any assertion fails or `isError: true`.

---

## 7. deckhouse_GetNodeGroup

**Call:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_GetNodeGroup '{"name":"master"}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
DATA=$(echo "$RESP" | jq -r '.result.content[0].text | fromjson')
echo "$DATA" | jq -e '.name == "master"'
echo "$DATA" | jq -e '.nodeType == "CloudEphemeral"'
echo "$DATA" | jq -e '.ready == 1 and .total == 1'
echo "$DATA" | jq -e '.nodeNames | type == "array"'
```

**Also test non-existent group:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_GetNodeGroup '{"name":"nonexistent"}')
echo "$RESP" | jq -e '.result.isError == true'
echo "$RESP" | jq -r '.result.content[0].text' | grep -qi "not found"
```

**PASS:** First call succeeds with correct data; second call returns `isError: true` with "not found".
**FAIL:** Wrong data, missing fields, or unexpected success on non-existent group.

---

## 8. deckhouse_GetDeckhouseLogs

**Expected: FAIL (no Deckhouse pod in Kind)**

```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_GetDeckhouseLogs '{"tail":50}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError == true'
echo "$RESP" | jq -r '.result.content[0].text' | grep -qi "deckhouse pod not found"
```

**PASS:** `isError: true` and message contains "deckhouse pod not found".
**FAIL:** No error (would mean a pod exists unexpectedly) or wrong error message.

---

## 9. deckhouse_GetNodeEvents

**Note:** Uses the same node name obtained in test 6.

```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_GetNodeEvents "{\"name\":\"$NODE_NAME\"}")
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
# events is an array (empty or populated)
echo "$RESP" | jq -r '.result.content[0].text | fromjson | .events | type == "array"'
```

**PASS:** No error; `events` is an array.
**FAIL:** `isError: true`.

---

## 10. deckhouse_GetStaticInstance

**Expected: FAIL (no static instances in Kind)**

```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_GetStaticInstance '{"name":"nonexistent-instance"}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError == true'
echo "$RESP" | jq -r '.result.content[0].text' | grep -qi "not found"
```

**PASS:** `isError: true` and message contains "not found".
**FAIL:** Unexpected success or wrong error.

---

## 11. deckhouse_GetPodLogs

**Expected: FAIL (test with a non-existent pod)**

```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_GetPodLogs '{"namespace":"d8-system","pod":"deckhouse-nonexistent","tail":10}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError == true'
echo "$RESP" | jq -r '.result.content[0].text' | grep -qi "not found"
```

**Also test with a real pod (kube-apiserver):**
```bash
# Get the actual kube-apiserver pod name
API_POD=$(kubectl get pod -n kube-system -l component=kube-apiserver -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [[ -n "$API_POD" ]]; then
  RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
    deckhouse_GetPodLogs "{\"namespace\":\"kube-system\",\"pod\":\"$API_POD\",\"container\":\"kube-apiserver\",\"tail\":5}")
  echo "$RESP" | jq -e '.result.isError // false | not'
  echo "$RESP" | jq -r '.result.content[0].text | fromjson | .logs | length >= 0'
fi
```

**PASS:** Non-existent pod → `isError: true` with "not found". Real pod (if found) → success with logs string.
**FAIL:** Wrong behavior on either sub-case.
