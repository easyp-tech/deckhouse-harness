---
name: e2e-test-deckhouse-harness
version: 1.0.0
description: |
  Full end-to-end manual testing of the deckhouse-harness MCP server over JSON-RPC against a local Kind
  cluster. Use when asked: "e2e test", "test mcp server", "run e2e", "verify mcp", "check all tools",
  "протестировать сервер", "проверить mcp", "запустить e2e".
---

# E2E Test Skill — Deckhouse MCP Server

## Overview

This skill drives a full end-to-end test of **all 42 MCP tools** exposed by the `deckhouse-harness` server.
The server is launched as a local process and exercised via **JSON-RPC 2.0 over stdin/stdout** — the same
transport used by Claude Desktop, Cursor, and other MCP clients.

| Item | Detail |
|------|--------|
| Environment | Local Kind cluster (`tests/integration/`) |
| Transport | stdio JSON-RPC — one-shot binary per tool call |
| Coverage | 42 tools across 6 API groups |
| Test data | Fixtures in `tests/integration/fixtures.yaml` |
| Output | PASS / FAIL table printed to chat |

---

## Prerequisites

The following tools must be installed and on `$PATH` before starting:

- `kind` — Kubernetes in Docker
- `kubectl` — Kubernetes CLI
- `go` — Go toolchain (1.26+)
- `jq` — JSON processor (for assertions)

---

## Procedure

### Step 1 — Set up the cluster

```bash
bash .agents/skills/e2e-test-deckhouse-harness/scripts/setup.sh
```

This script:
1. Checks prerequisites
2. Runs `tests/integration/setup.sh` — creates a Kind cluster named `d8`, waits for CRDs (~15 min first run, instant on reuse)
3. Applies `tests/integration/fixtures.yaml` — seeds NodeGroups, ModuleConfigs, DeckhouseReleases
4. Builds the binary to `tests/integration/deckhouse-harness`

On success it prints the `KUBECONFIG` path. **Export that path** for subsequent steps:
```bash
export KUBECONFIG=<path printed by setup>
```

### Step 2 — Run tool groups in order

Execute each test-cases reference file in this order to avoid write-test interference:

| Order | Group | Reference file | Tools |
|-------|-------|---------------|-------|
| 1 | DiagnosticsAPI | `references/test-cases-diagnostics.md` | 11 |
| 2 | ReleasesAPI | `references/test-cases-releases.md` | 3 |
| 3 | ModulesAPI | `references/test-cases-modules.md` | 7 |
| 4 | ConfigAPI | `references/test-cases-config.md` | 3 |
| 5 | SourcesAPI | `references/test-cases-sources.md` | 6 |
| 6 | NodesAPI | `references/test-cases-nodes.md` | 13 |

For each tool case: run the bash call, parse the response, evaluate the assertions, record PASS or FAIL.

### Step 3 — Teardown

```bash
bash .agents/skills/e2e-test-deckhouse-harness/scripts/teardown.sh
```

Deletes all test resources labelled `e2e-test=true`. The Kind cluster itself is **not** deleted.

### Step 4 — Print report

After all 42 tools, print a summary table:

```
| Tool                              | Status | Notes                        |
|-----------------------------------|--------|------------------------------|
| deckhouse_GetClusterStatus        | PASS   |                              |
| deckhouse_ListNodes               | PASS   |                              |
| deckhouse_GetDeckhouseLogs        | PASS   | expected error, correct msg  |
| ...                               | ...    |                              |
```

Final line: `TOTAL: N/42 PASS` (where expected failures count as PASS when error message matches).

---

## How to Call a Tool

Use the helper script for every tool invocation:

```bash
bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  <tool_name> '<args_json>'
```

**Examples:**

```bash
# No arguments (empty object)
bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_GetClusterStatus '{}'

# With arguments
bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_GetModuleConfig '{"name":"prometheus"}'

# With enum value (ProtoJSON: use the string name)
bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_ListNodes '{"status":"NODE_STATUS_FILTER_READY"}'
```

**Parsing the response:**

The script prints one JSON line containing the full JSON-RPC response for the tool call. Use `jq`:

```bash
bash .agents/skills/e2e-test-deckhouse-harness/scripts/mcp-call.sh \
  deckhouse_GetClusterStatus '{}' \
  | jq '.result.content[0].text | fromjson | .nodes.total'
```

---

## PASS / FAIL Criteria

**PASS conditions:**
- Response has no `"isError": true` at the top level AND all listed field assertions hold.
- OR the tool is expected to fail in Kind (see table below) AND the error message matches the expected pattern.

**FAIL conditions:**
- Unexpected `"isError": true`.
- Expected field is absent or has wrong value.
- The script exits non-zero (binary crash / connection error).
- For expected-failure tools: wrong error message or no error at all.

---

## Known Limitations in Kind (Expected Failures)

These tools behave differently in Kind because there is no real Deckhouse operator running.
They are **still tested** — the test verifies the error message is correct.

| Tool | Kind behavior | Expected error contains |
|------|--------------|------------------------|
| `deckhouse_GetClusterConfiguration` | Secret `kube-system/d8-cluster-configuration` absent | `"not found"` |
| `deckhouse_GetStaticClusterConfiguration` | Same secret absent | `"not found"` |
| `deckhouse_UpdateKubernetesVersion` | Same secret absent | `"not found"` |
| `deckhouse_GetDeckhouseLogs` | No pod with `app=deckhouse` in `d8-system` | `"deckhouse pod not found"` |
| `deckhouse_GetPodLogs` | Test uses non-existent pod | `"not found"` |
| `deckhouse_GetStaticInstance` | No StaticInstances exist | `"not found"` |
| `deckhouse_WaitNodeReady` | No static nodes bootstrapped | timeout response (not error) |
| `deckhouse_DrainNode` | Called on non-existent node | `"not found"` |
| `deckhouse_RemoveNode` | Kind node is not a static node | `"not found"` |
| `deckhouse_ListModules` | No Module CRs in Kind | empty list (not error) |
| `deckhouse_ListModuleReleases` | No ModuleRelease CRs | empty list (not error) |

All other tools are expected to **succeed** (return data without `isError`).

---

## Cleanup Labelling Convention

All resources created during write tests **must** be labelled immediately after creation:

```bash
kubectl label <resource-type> <name> e2e-test=true
```

This allows `teardown.sh` to find and delete them on a failed run.
Each test case in the reference files includes the exact label command.
