# Test Cases — SourcesAPI (6 tools)

These tests follow a create → verify → delete → verify-gone pattern.
All created resources are labelled `e2e-test=true` immediately after creation via `kubectl label`.

No fixtures are seeded for this group — everything starts empty.

---

## 1. deckhouse_ListModuleSources

**Call (expect empty list before any sources are created):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListModuleSources '{}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
echo "$RESP" | jq -r '.result.content[0].text | fromjson | .sources | type == "array"'
```

**PASS:** No error; `sources` is an array (empty is fine).
**FAIL:** `isError: true`.

---

## 2. deckhouse_CreateModuleSource

**Create a test source:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_CreateModuleSource \
  '{"name":"e2e-test-source","registry":"registry.example.com/deckhouse-modules"}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
DATA=$(echo "$RESP" | jq -r '.result.content[0].text | fromjson')
echo "$DATA" | jq -e '.name == "e2e-test-source"'

# Add cleanup label
kubectl label modulesource e2e-test-source e2e-test=true
```

**Verify via ListModuleSources:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListModuleSources '{}')
echo "$RESP" | jq -r '.result.content[0].text | fromjson | [.sources[].name] | contains(["e2e-test-source"])'
```

**PASS:** Source created, appears in list.
**FAIL:** `isError: true` or source not in list.

---

## 3. deckhouse_ListModuleUpdatePolicies

**Call (expect empty list before any policies are created):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListModuleUpdatePolicies '{}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
echo "$RESP" | jq -r '.result.content[0].text | fromjson | .policies | type == "array"'
```

**PASS:** No error; `policies` is an array.
**FAIL:** `isError: true`.

---

## 4. deckhouse_CreateModuleUpdatePolicy

**Create a test update policy:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_CreateModuleUpdatePolicy \
  '{"name":"e2e-test-policy","update_mode":"Auto","match_labels":{"source":"e2e-test-source"}}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
DATA=$(echo "$RESP" | jq -r '.result.content[0].text | fromjson')
echo "$DATA" | jq -e '.name == "e2e-test-policy"'

# Add cleanup label
kubectl label moduleupdatepolicy e2e-test-policy e2e-test=true

# Verify via list
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListModuleUpdatePolicies '{}')
echo "$RESP" | jq -r '.result.content[0].text | fromjson | [.policies[].name] | contains(["e2e-test-policy"])'
```

**Test invalid (missing match_labels):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_CreateModuleUpdatePolicy \
  '{"name":"e2e-test-invalid","update_mode":"Auto","match_labels":{}}')
echo "$RESP" | jq -e '.result.isError == true'
echo "$RESP" | jq -r '.result.content[0].text' | grep -qi "match_labels"
```

**PASS:** Policy created and appears in list; empty match_labels returns error mentioning "match_labels".
**FAIL:** Any unexpected behavior.

---

## 5. deckhouse_ListModuleReleases

**Expected: empty list (no ModuleRelease CRs in Kind)**

```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListModuleReleases '{"module_name":"prometheus"}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
echo "$RESP" | jq -r '.result.content[0].text | fromjson | .releases | type == "array"'
# Empty is expected
echo "$RESP" | jq -r '.result.content[0].text | fromjson | .releases | length == 0'
```

**Test missing module_name (validation error):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListModuleReleases '{}')
echo "$RESP" | jq -e '.result.isError == true'
echo "$RESP" | jq -r '.result.content[0].text' | grep -qi "module_name"
```

**PASS:** With module_name returns empty array without error; without module_name returns error.
**FAIL:** Unexpected error on valid call, or missing validation on empty module_name.

---

## 6. deckhouse_DeleteModuleSource

**Delete the source created in test 2 (no active releases → should succeed):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_DeleteModuleSource '{"name":"e2e-test-source","force":false}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'

# Verify it is gone
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListModuleSources '{}')
echo "$RESP" | jq -r '.result.content[0].text | fromjson | [.sources[].name] | contains(["e2e-test-source"]) | not'
```

**Delete non-existent source:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_DeleteModuleSource '{"name":"nonexistent-source"}')
echo "$RESP" | jq -e '.result.isError == true'
echo "$RESP" | jq -r '.result.content[0].text' | grep -qi "not found"
```

**Also clean up the update policy:**
```bash
bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_CreateModuleUpdatePolicy \
  '{"name":"dummy","update_mode":"Manual","match_labels":{"x":"y"}}' > /dev/null 2>&1 || true
kubectl delete moduleupdatepolicy e2e-test-policy --ignore-not-found
```

**PASS:** Source deleted and not in list; non-existent delete returns error.
**FAIL:** Source not deleted or wrong error on non-existent.
