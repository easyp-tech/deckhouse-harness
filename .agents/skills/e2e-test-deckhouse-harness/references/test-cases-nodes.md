# Test Cases — NodesAPI (13 tools)

All created resources are labelled `e2e-test=true` immediately after creation.
Teardown order: delete StaticInstances/SSHCredentials before NodeGroups (dependency).

Tests are grouped by sub-flow. Run them in the order listed.

---

## Sub-flow A: NodeGroup lifecycle

### 1. deckhouse_CreateNodeGroup

**Create a test NodeGroup:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_CreateNodeGroup \
  '{"name":"e2e-test-workers","node_type":"Static"}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
DATA=$(echo "$RESP" | jq -r '.result.content[0].text | fromjson')
echo "$DATA" | jq -e '.name == "e2e-test-workers"'

# Add cleanup label
kubectl label nodegroup e2e-test-workers e2e-test=true

# Verify via ListNodeGroups
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListNodeGroups '{}')
echo "$RESP" | jq -r '.result.content[0].text | fromjson | [.nodeGroups[].name] | contains(["e2e-test-workers"])'
```

**Duplicate creation should fail:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_CreateNodeGroup \
  '{"name":"e2e-test-workers","node_type":"Static"}')
echo "$RESP" | jq -e '.result.isError == true'
echo "$RESP" | jq -r '.result.content[0].text' | grep -qi "already"
```

**PASS:** NodeGroup created, appears in list; duplicate returns error with "already".
**FAIL:** Any unexpected behavior.

---

### 2. deckhouse_DeleteNodeGroup

**Delete the NodeGroup (after all StaticInstances in it are removed):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_DeleteNodeGroup '{"name":"e2e-test-workers"}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'

# Verify removed
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListNodeGroups '{}')
echo "$RESP" | jq -r '.result.content[0].text | fromjson | [.nodeGroups[].name] | contains(["e2e-test-workers"]) | not'
```

**Non-existent NodeGroup:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_DeleteNodeGroup '{"name":"nonexistent-group"}')
echo "$RESP" | jq -e '.result.isError == true'
echo "$RESP" | jq -r '.result.content[0].text' | grep -qi "not found"
```

**PASS:** NodeGroup deleted and not in list; non-existent returns error.
**FAIL:** Any unexpected behavior.

---

## Sub-flow B: SSHCredentials + StaticInstance lifecycle

### 3. deckhouse_CreateSSHCredentials

```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_CreateSSHCredentials \
  '{"name":"e2e-test-creds","user":"ubuntu","private_key":"LS0tLS1CRUdJTiBPUEVOU1NIIFBSSVZBVEUgS0VZLS0tLS0K"}')
```

Note: `private_key` is a Base64-encoded dummy key (the handler re-encodes it before storing).
You can use any Base64 string: `echo -n "fake-key" | base64` → `ZmFrZS1rZXk=`

```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_CreateSSHCredentials \
  '{"name":"e2e-test-creds","user":"ubuntu","private_key":"ZmFrZS1rZXk="}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
echo "$RESP" | jq -r '.result.content[0].text | fromjson | .name == "e2e-test-creds"'

kubectl label sshcredentials e2e-test-creds e2e-test=true
```

**Missing private_key validation:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_CreateSSHCredentials '{"name":"bad-creds","user":"ubuntu","private_key":""}')
echo "$RESP" | jq -e '.result.isError == true'
```

**PASS:** Credentials created; empty key returns error.
**FAIL:** Any unexpected behavior.

---

### 4. deckhouse_CreateStaticInstance

```bash
# First ensure e2e-test-workers NodeGroup exists (re-create if deleted above)
bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_CreateNodeGroup '{"name":"e2e-test-workers","node_type":"Static"}' > /dev/null 2>&1 || true
kubectl label nodegroup e2e-test-workers e2e-test=true 2>/dev/null || true

RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_CreateStaticInstance \
  '{"name":"e2e-test-node-01","address":"192.0.2.1","credentials_ref":"e2e-test-creds","labels":{"node.deckhouse.io/group":"e2e-test-workers"}}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
echo "$RESP" | jq -r '.result.content[0].text | fromjson | .name == "e2e-test-node-01"'

kubectl label staticinstance e2e-test-node-01 e2e-test=true

# Verify via ListStaticInstances
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListStaticInstances '{}')
echo "$RESP" | jq -r '.result.content[0].text | fromjson | [.instances[].name] | contains(["e2e-test-node-01"])'
```

**PASS:** StaticInstance created and appears in list.
**FAIL:** `isError: true` or not in list.

---

### 5. deckhouse_DeleteStaticInstance

```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_DeleteStaticInstance '{"name":"e2e-test-node-01"}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
echo "$RESP" | jq -r '.result.content[0].text | fromjson | .success == true'

RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListStaticInstances '{}')
echo "$RESP" | jq -r '.result.content[0].text | fromjson | [.instances[].name] | contains(["e2e-test-node-01"]) | not'
```

**PASS:** Deleted successfully; not in list after.
**FAIL:** `isError: true` or still in list.

---

### 6. deckhouse_DeleteSSHCredentials

```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_DeleteSSHCredentials '{"name":"e2e-test-creds"}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'

kubectl get sshcredentials e2e-test-creds 2>&1 | grep -qi "not found"
```

**PASS:** Deleted; not found via kubectl.
**FAIL:** `isError: true` or resource still exists.

---

## Sub-flow C: AddWorkerNode composite

### 7. deckhouse_AddWorkerNode

**Test with `wait_ready: false` (Kind cannot bootstrap a real node):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_AddWorkerNode \
  '{"address":"192.0.2.99","ssh_user":"ubuntu","private_key":"ZmFrZS1rZXk=","node_group":"e2e-test-workers","node_name":"e2e-add-worker","wait_ready":false}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
DATA=$(echo "$RESP" | jq -r '.result.content[0].text | fromjson')
echo "$DATA" | jq -e '.sshCredentialsName == "e2e-add-worker-creds"'
echo "$DATA" | jq -e '.staticInstanceName == "e2e-add-worker"'

# Label created resources for cleanup
kubectl label sshcredentials e2e-add-worker-creds e2e-test=true 2>/dev/null || true
kubectl label staticinstance e2e-add-worker e2e-test=true 2>/dev/null || true
```

**Cleanup:**
```bash
bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_DeleteStaticInstance '{"name":"e2e-add-worker"}' > /dev/null
bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_DeleteSSHCredentials '{"name":"e2e-add-worker-creds"}' > /dev/null
```

**PASS:** Creates both SSHCredentials and StaticInstance; returns their names.
**FAIL:** `isError: true` or wrong resource names.

---

## Sub-flow D: WaitNodeReady

### 8. deckhouse_WaitNodeReady

**Expected: timeout response (no real Deckhouse to bootstrap)**

```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_WaitNodeReady \
  '{"name":"nonexistent-static-node","timeout_seconds":30,"interval_seconds":5}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
DATA=$(echo "$RESP" | jq -r '.result.content[0].text | fromjson')
# Should return timed_out: true (not an error, just a timeout state)
echo "$DATA" | jq -e '.timedOut == true'
```

**PASS:** Returns `timedOut: true` without `isError`.
**FAIL:** `isError: true` or `timedOut: false`.

---

## Sub-flow E: Cordon / Uncordon / Drain

> Note: Uses the Kind control-plane node. Cordon/uncordon are safe on a test cluster.
> Get the node name first from `deckhouse_ListNodes` (stored in `$NODE_NAME`).

### 9. deckhouse_CordonNode

```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_CordonNode "{\"name\":\"$NODE_NAME\"}")
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
DATA=$(echo "$RESP" | jq -r '.result.content[0].text | fromjson')
# previous_state was false (node was schedulable)
echo "$DATA" | jq -e '.previousState == false'

# Verify node is now unschedulable
kubectl get node "$NODE_NAME" -o jsonpath='{.spec.unschedulable}' | grep -q "true"
```

**Idempotency:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_CordonNode "{\"name\":\"$NODE_NAME\"}")
echo "$RESP" | jq -e '.result.isError // false | not'
echo "$RESP" | jq -r '.result.content[0].text | fromjson | .previousState == true'
```

**PASS:** Node becomes unschedulable; second call returns `previousState: true`.
**FAIL:** Wrong previous state or `isError: true`.

---

### 10. deckhouse_UncordonNode

```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_UncordonNode "{\"name\":\"$NODE_NAME\"}")
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
echo "$RESP" | jq -r '.result.content[0].text | fromjson | .previousState == true'

# Verify node is now schedulable
UNSCHED=$(kubectl get node "$NODE_NAME" -o jsonpath='{.spec.unschedulable}' 2>/dev/null || echo "")
[[ -z "$UNSCHED" || "$UNSCHED" == "false" ]]
```

**PASS:** Node becomes schedulable again; `previousState: true`.
**FAIL:** Node still unschedulable or wrong previous state.

---

### 11. deckhouse_DrainNode

**Expected: FAIL (call on a non-existent node name)**

```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_DrainNode '{"name":"nonexistent-node","timeout_seconds":30}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError == true'
echo "$RESP" | jq -r '.result.content[0].text' | grep -qi "not found"
```

**PASS:** `isError: true` with "not found" in message.
**FAIL:** Unexpected success.

---

## Sub-flow F: RemoveNode

### 12. deckhouse_RemoveNode

**Expected: FAIL (Kind control-plane is not a static node — no StaticInstance exists for it)**

```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_RemoveNode "{\"name\":\"$NODE_NAME\",\"drain\":false}")
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError == true'
echo "$RESP" | jq -r '.result.content[0].text' | grep -qi "not found"
```

**PASS:** `isError: true` with "not found" (no StaticInstance for this node).
**FAIL:** Unexpected success or wrong error.

---

## Sub-flow G: NodeGroupConfiguration

### 13. deckhouse_CreateNodeGroupConfiguration

```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_CreateNodeGroupConfiguration \
  '{"name":"e2e-test-ngconfig","content":"#!/bin/bash\necho hello > /tmp/e2e.txt","node_groups":["e2e-test-workers"],"weight":50}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
echo "$RESP" | jq -r '.result.content[0].text | fromjson | .name == "e2e-test-ngconfig"'

kubectl label nodegroupconfiguration e2e-test-ngconfig e2e-test=true

kubectl get nodegroupconfiguration e2e-test-ngconfig \
  -o jsonpath='{.spec.nodeGroupSelector.matchNames[0]}' | grep -q "e2e-test-workers"
```

**Validation: empty content should fail:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_CreateNodeGroupConfiguration \
  '{"name":"bad-ngconfig","content":"","node_groups":["worker"]}')
echo "$RESP" | jq -e '.result.isError == true'
```

**Validation: empty node_groups should fail:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_CreateNodeGroupConfiguration \
  '{"name":"bad-ngconfig2","content":"#!/bin/bash\necho hi","node_groups":[]}')
echo "$RESP" | jq -e '.result.isError == true'
```

**Cleanup:**
```bash
kubectl delete nodegroupconfiguration e2e-test-ngconfig --ignore-not-found
kubectl delete nodegroup e2e-test-workers --ignore-not-found
```

**PASS:** Config created with correct spec; both validation cases return errors.
**FAIL:** Any unexpected behavior.
