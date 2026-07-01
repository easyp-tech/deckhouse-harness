# Test Cases — ReleasesAPI (3 tools)

Fixtures:
- `v1.70.0` — phase: Deployed, `status.approved: true`
- `v1.71.0` — phase: Pending, `status.approved: false`

---

## 1. deckhouse_ListDeckhouseReleases

**Call (no filter — all releases):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListDeckhouseReleases '{}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
DATA=$(echo "$RESP" | jq -r '.result.content[0].text | fromjson')
echo "$DATA" | jq -e '.releases | length == 2'
echo "$DATA" | jq -e '[.releases[].name] | contains(["v1.70.0","v1.71.0"])'
```

**Call (filter by PENDING phase):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListDeckhouseReleases '{"phase":"DECKHOUSE_RELEASE_PHASE_PENDING"}')
echo "$RESP" | jq -e '.result.isError // false | not'
DATA=$(echo "$RESP" | jq -r '.result.content[0].text | fromjson')
echo "$DATA" | jq -e '.releases | length == 1'
echo "$DATA" | jq -e '.releases[0].name == "v1.71.0"'
```

**Call (filter by DEPLOYED phase):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListDeckhouseReleases '{"phase":"DECKHOUSE_RELEASE_PHASE_DEPLOYED"}')
echo "$RESP" | jq -e '.result.isError // false | not'
DATA=$(echo "$RESP" | jq -r '.result.content[0].text | fromjson')
echo "$DATA" | jq -e '.releases | length == 1'
echo "$DATA" | jq -e '.releases[0].name == "v1.70.0"'
```

**PASS:** All three calls return correct filtered lists without error.
**FAIL:** Wrong counts, wrong names, or `isError: true`.

---

## 2. deckhouse_GetDeckhouseRelease

**Call (existing deployed release):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_GetDeckhouseRelease '{"version":"v1.70.0"}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
DATA=$(echo "$RESP" | jq -r '.result.content[0].text | fromjson')
echo "$DATA" | jq -e '.name == "v1.70.0"'
echo "$DATA" | jq -e '.phase == "Deployed"'
echo "$DATA" | jq -e '.approved == true'
```

**Call (pending release):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_GetDeckhouseRelease '{"version":"v1.71.0"}')
echo "$RESP" | jq -e '.result.isError // false | not'
DATA=$(echo "$RESP" | jq -r '.result.content[0].text | fromjson')
echo "$DATA" | jq -e '.name == "v1.71.0"'
echo "$DATA" | jq -e '.phase == "Pending"'
echo "$DATA" | jq -e '.approved == false'
```

**Call (non-existent release):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_GetDeckhouseRelease '{"version":"v9.99.0"}')
echo "$RESP" | jq -e '.result.isError == true'
echo "$RESP" | jq -r '.result.content[0].text' | grep -qi "not found"
```

**PASS:** First two succeed with correct data; third returns `isError: true` with "not found".
**FAIL:** Any mismatch.

---

## 3. deckhouse_ApproveRelease

**Call (approve the pending release v1.71.0):**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ApproveRelease '{"version":"v1.71.0"}')
```

**Assertions:**
```bash
echo "$RESP" | jq -e '.result.isError // false | not'
DATA=$(echo "$RESP" | jq -r '.result.content[0].text | fromjson')
# previous_approved should be false (was not approved before)
echo "$DATA" | jq -e '.previousApproved == false'
```

**Verify the annotation was applied:**
```bash
kubectl get deckhouserelease v1.71.0 \
  -o jsonpath='{.metadata.annotations.release\.deckhouse\.io/approved}' \
  | grep -q "true"
```

**Idempotency — call again, should succeed:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ApproveRelease '{"version":"v1.71.0"}')
echo "$RESP" | jq -e '.result.isError // false | not'
# Now previous_approved is true (already approved)
echo "$RESP" | jq -r '.result.content[0].text | fromjson | .previousApproved == true'
```

**Call on non-existent release:**
```bash
RESP=$(bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ApproveRelease '{"version":"v9.99.0"}')
echo "$RESP" | jq -e '.result.isError == true'
```

**PASS:** Approve succeeds, annotation appears on the resource, idempotent call succeeds, non-existent returns error.
**FAIL:** Any unexpected behavior.
