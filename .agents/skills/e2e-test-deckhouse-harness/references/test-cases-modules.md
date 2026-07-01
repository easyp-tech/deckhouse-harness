# Test Cases — ModulesAPI (7 tools)

Fixtures (from `tests/integration/fixtures.yaml`):
- Enabled: `deckhouse`, `global`, `prometheus`, `ingress-nginx`, `cert-manager`
- Disabled: `node-manager`

**State restoration:** Tests that mutate module state (enable/disable/settings) restore original state
before moving to the next test.

---

## 1. deckhouse_ListModuleConfigs

**Call (no filter — all configs):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListModuleConfigs '{}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
DATA=$(echo "$RESP" | jq -r '.result.content[0].text | fromjson')
echo "$DATA" | jq -e '.moduleConfigs | length == 6'
NAMES=$(echo "$DATA" | jq -r '[.moduleConfigs[].name] | sort | @csv')
echo "$NAMES" | grep -q "deckhouse"
echo "$NAMES" | grep -q "node-manager"
```

**Call (filter enabled=true):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListModuleConfigs '{"enabled":true}')
echo "$RESP" | jq -e '.result.isError // false | not'
DATA=$(echo "$RESP" | jq -r '.result.content[0].text | fromjson')
echo "$DATA" | jq -e '.moduleConfigs | length == 5'
# node-manager must NOT be in the list
echo "$DATA" | jq -e '[.moduleConfigs[].name] | contains(["node-manager"]) | not'
```

**Call (filter enabled=false):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListModuleConfigs '{"enabled":false}')
echo "$RESP" | jq -e '.result.isError // false | not'
DATA=$(echo "$RESP" | jq -r '.result.content[0].text | fromjson')
echo "$DATA" | jq -e '.moduleConfigs | length == 1'
echo "$DATA" | jq -e '.moduleConfigs[0].name == "node-manager"'
```

**PASS:** All three calls return correct filtered lists.
**FAIL:** Wrong counts, wrong names, or `isError: true`.

---

## 2. deckhouse_GetModuleConfig

**Call:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_GetModuleConfig '{"name":"deckhouse"}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
DATA=$(echo "$RESP" | jq -r '.result.content[0].text | fromjson')
echo "$DATA" | jq -e '.name == "deckhouse"'
echo "$DATA" | jq -e '.enabled == true'
echo "$DATA" | jq -e '.version == 1'
```

**Call (disabled module):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_GetModuleConfig '{"name":"node-manager"}')
echo "$RESP" | jq -e '.result.isError // false | not'
echo "$RESP" | jq -r '.result.content[0].text | fromjson | .enabled == false'
```

**Call (non-existent):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_GetModuleConfig '{"name":"nonexistent-module"}')
echo "$RESP" | jq -e '.result.isError == true'
echo "$RESP" | jq -r '.result.content[0].text' | grep -qi "not found"
```

**PASS:** Correct enabled state for each call; non-existent returns error.
**FAIL:** Wrong enabled value or missing fields.

---

## 3. deckhouse_EnableModule

**Enable `node-manager` (currently disabled in fixtures):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_EnableModule '{"name":"node-manager"}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
DATA=$(echo "$RESP" | jq -r '.result.content[0].text | fromjson')
echo "$DATA" | jq -e '.previousState == false'

# Verify via GetModuleConfig
RESP2=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_GetModuleConfig '{"name":"node-manager"}')
echo "$RESP2" | jq -r '.result.content[0].text | fromjson | .enabled == true'
```

**Idempotency:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_EnableModule '{"name":"node-manager"}')
echo "$RESP" | jq -e '.result.isError // false | not'
echo "$RESP" | jq -r '.result.content[0].text | fromjson | .previousState == true'
```

**Restore original state:**
```bash
bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_DisableModule '{"name":"node-manager"}' > /dev/null
```

**PASS:** Enable returns `previousState: false`; idempotent call returns `previousState: true`; verify succeeds.
**FAIL:** Wrong `previousState` or `isError: true`.

---

## 4. deckhouse_DisableModule

**Disable `cert-manager` (currently enabled in fixtures):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_DisableModule '{"name":"cert-manager"}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
echo "$RESP" | jq -r '.result.content[0].text | fromjson | .previousState == true'

# Verify
RESP2=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_GetModuleConfig '{"name":"cert-manager"}')
echo "$RESP2" | jq -r '.result.content[0].text | fromjson | .enabled == false'
```

**Restore:**
```bash
bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_EnableModule '{"name":"cert-manager"}' > /dev/null
```

**PASS:** Disable returns `previousState: true`; verify shows disabled.
**FAIL:** Wrong `previousState` or `isError: true`.

---

## 5. deckhouse_ListModules

**Expected: returns empty list (no Module CRs in Kind without real Deckhouse)**

```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListModules '{}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
# modules is an array (empty is valid)
echo "$RESP" | jq -r '.result.content[0].text | fromjson | .modules | type == "array"'
```

**PASS:** No error; `modules` is an array.
**FAIL:** `isError: true`.

---

## 6. deckhouse_UpdateModuleSettings

**Update `deckhouse` module settings — change releaseChannel:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_UpdateModuleSettings '{"name":"deckhouse","settings":{"releaseChannel":"Stable"}}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'

# Verify the setting was applied
RESP2=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_GetModuleConfig '{"name":"deckhouse"}')
echo "$RESP2" | jq -e '.result.isError // false | not'
# (Settings verification via kubectl since GetModuleConfig returns best-effort map)
kubectl get moduleconfig deckhouse \
  -o jsonpath='{.spec.settings.releaseChannel}' | grep -q "Stable"
```

**Restore original releaseChannel:**
```bash
bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_UpdateModuleSettings \
  '{"name":"deckhouse","settings":{"releaseChannel":"EarlyAccess"}}' > /dev/null
```

**Call with empty settings (should error):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_UpdateModuleSettings '{"name":"deckhouse","settings":{}}')
echo "$RESP" | jq -e '.result.isError == true'
```

**PASS:** Settings applied (verified via kubectl); empty settings returns error.
**FAIL:** Settings not applied or wrong error behavior.

---

## 7. deckhouse_SetModuleMaintenance

**Enable maintenance on `prometheus`:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_SetModuleMaintenance '{"name":"prometheus","enabled":true}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'

# Verify spec.maintenance was set
kubectl get moduleconfig prometheus \
  -o jsonpath='{.spec.maintenance}' | grep -q "NoResourceReconciliation"
```

**Disable maintenance (restore):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_SetModuleMaintenance '{"name":"prometheus","enabled":false}')
echo "$RESP" | jq -e '.result.isError // false | not'

# Verify field was removed
MAINT=$(kubectl get moduleconfig prometheus \
  -o jsonpath='{.spec.maintenance}' 2>/dev/null || echo "")
echo "Maintenance after disable: '$MAINT' (expected empty)"
[[ -z "$MAINT" ]]
```

**PASS:** Enable sets `spec.maintenance=NoResourceReconciliation`; disable removes the field.
**FAIL:** Field not set/removed correctly or `isError: true`.
