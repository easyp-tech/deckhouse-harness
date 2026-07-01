# Test Cases — ConfigAPI (3 tools)

**All three tools are expected to FAIL in Kind** because the Secret
`kube-system/d8-cluster-configuration` does not exist in a bare Kind cluster.
The tests verify that errors are handled correctly — the right error message is returned.

---

## 1. deckhouse_GetClusterConfiguration

**Expected: FAIL (Secret not found)**

```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_GetClusterConfiguration '{}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError == true'
echo "$RESP" | jq -r '.result.content[0].text' | grep -qi "not found"
```

**PASS:** `isError: true` and error text contains "not found".
**FAIL:** Tool succeeds (Secret unexpectedly exists), or error message does not match.

---

## 2. deckhouse_GetStaticClusterConfiguration

**Expected: FAIL (same Secret not found)**

```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_GetStaticClusterConfiguration '{}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError == true'
echo "$RESP" | jq -r '.result.content[0].text' | grep -qi "not found"
```

**PASS:** `isError: true` and error text contains "not found".
**FAIL:** Unexpected success or wrong error message.

---

## 3. deckhouse_UpdateKubernetesVersion

**Expected: FAIL (Secret not found, cannot read current configuration)**

```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_UpdateKubernetesVersion '{"version":"1.29"}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError == true'
echo "$RESP" | jq -r '.result.content[0].text' | grep -qi "not found"
```

**Also test invalid version format (validation should reject before K8s call):**
```bash
# "1.29.0" has 3 components — pattern requires exactly ^[0-9]+\.[0-9]+$
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_UpdateKubernetesVersion '{"version":"1.29.0"}')
# Should also fail (format validation)
echo "$RESP" | jq -e '.result.isError == true'
```

**PASS:** Both calls return `isError: true`. Valid format returns "not found"; invalid format returns validation error.
**FAIL:** Either call unexpectedly succeeds.

---

## Note: Testing ConfigAPI with a Real Cluster

If `kube-system/d8-cluster-configuration` Secret exists (e.g., on a real Deckhouse cluster), these tools
would succeed. In that context, test assertions should flip:

- `GetClusterConfiguration` → PASS, `yaml` field is non-empty YAML string
- `GetStaticClusterConfiguration` → PASS or FAIL depending on whether static config exists
- `UpdateKubernetesVersion` → PASS, `previousVersion` matches current k8s version, verify with kubectl
