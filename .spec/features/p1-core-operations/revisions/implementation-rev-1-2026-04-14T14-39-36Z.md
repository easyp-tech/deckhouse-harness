# P1 Core Operations — Implementation Report

## Summary

All 8 tasks completed. The project builds cleanly and all tests pass.

## Tasks

### T-1: k8s.Client +11 methods + mockClient ✅

`internal/k8s/client.go` extended with 11 new methods:
- `GetNode`, `CordonNode` — node read/write
- `GetPodLogs` — streaming pod logs to string
- `GetSecret` — read core Secret (for ClusterConfiguration)
- `GetNodeGroup`, `CreateNodeGroup` — NodeGroup CRD
- `DeleteStaticInstance` — StaticInstance deletion
- `GetModuleConfig`, `UpdateModuleConfig` — ModuleConfig CRUD
- `GetDeckhouseRelease`, `PatchDeckhouseRelease` — DeckhouseRelease read+patch

`internal/handler/mock_client_test.go` extended with 11 corresponding function fields.

### T-2: Proto +13 RPCs + task generate ✅

Added RPCs to 5 proto files:
- `diagnostics.proto`: `GetNode`, `GetNodeGroup`, `GetDeckhouseLogs`
- `modules.proto`: `GetModuleConfig`, `EnableModule`, `DisableModule`
- `releases.proto`: `GetDeckhouseRelease`, `ApproveRelease`
- `nodes.proto`: `DeleteStaticInstance`, `RemoveNode`, `CreateNodeGroup`, `WaitNodeReady`
- `config.proto`: `GetClusterConfiguration` (first real RPC, replacing stub)

Generated files updated: `diagnostics.mcp.go`, `modules.mcp.go`, `releases.mcp.go`, `nodes.mcp.go`, `config.mcp.go` (new file).

### T-3: Tests for all 13 new handlers ✅

Added tests to existing files + created `config_test.go`:
- `diagnostics_test.go`: 8 tests (GetNode ×3, GetNodeGroup ×2, GetDeckhouseLogs ×3)
- `modules_test.go`: 7 tests (GetModuleConfig ×2, Enable ×3, Disable ×2)
- `releases_test.go`: 5 tests (GetDeckhouseRelease ×2, ApproveRelease ×3)
- `nodes_test.go`: 9 tests (DeleteStaticInstance ×2, RemoveNode ×3, CreateNodeGroup ×2, WaitNodeReady ×2)
- `config_test.go`: 2 tests (GetClusterConfiguration ×2)

Total: **31 new tests** in addition to existing 38.

### T-4: G1 GET handlers ✅

Implemented in existing handler files:
- `diagnostics.go`: `GetNode`, `GetNodeGroup`, `GetDeckhouseLogs`
- `modules.go`: `GetModuleConfig`
- `releases.go`: `GetDeckhouseRelease`

### T-5: G3 Simple write handlers ✅

- `nodes.go`: `DeleteStaticInstance`, `RemoveNode`
- `modules.go`: `EnableModule`, `DisableModule` (via shared `setModuleEnabled`)
- `releases.go`: `ApproveRelease` (merge patch on approval annotation)

### T-6: G4+G5 Composite + Config handlers ✅

- `nodes.go`:
  - `CreateNodeGroup` — creates NodeGroup CRD with spec from request
  - `WaitNodeReady` — extracted shared `pollStaticInstance` helper, delegates to it
  - ADR-1 implemented: `pollStaticInstance` extracted from `AddWorkerNode`, now reused by `WaitNodeReady`
- `config.go` (new file): `ConfigHandler.GetClusterConfiguration` — reads `d8-cluster-configuration` Secret from `kube-system`

### T-7: RBAC ✅

Updated `deploy/rbac.yaml`:
- Added `secrets` get for `d8-cluster-configuration` (resourceNames scoped)
- Added `nodes` update/patch for `CordonNode`
- Added `staticinstances` delete
- Added `nodegroups` create
- Added `moduleconfigs` update
- Added `deckhouserelease` patch

### T-8: GATE ✅

- `go build ./...` → **PASS**
- `go test ./internal/handler/... -timeout=300s` → **PASS** (121s, includes 4 polling tests ×30s each)
- Also registered `cmd/deckhouse-mcp/main.go`: `RegisterConfigAPITools` call added

## Architecture Decisions

| ADR | Decision |
|-----|---------|
| ADR-1 | `pollStaticInstance` extracted from `AddWorkerNode`, reused by `WaitNodeReady` |
| ADR-2 | `RemoveNode` = GetStaticInstance check → cordon → DeleteStaticInstance |
| ADR-3 | Enable/DisableModule = Get → modify `spec.enabled` → full Update |
| ADR-4 | `ApproveRelease` = merge patch `{"metadata":{"annotations":{"release.deckhouse.io/approved":"true"}}}` |
| ADR-5 | `GetClusterConfiguration` = GetSecret(`kube-system`, `d8-cluster-configuration`), key `cluster-configuration.yaml` |
| ADR-6 | `GetDeckhouseLogs` = ListPods(`d8-system`) → find `app=deckhouse` pod → GetPodLogs → client-side grep |

## Files Changed

| File | Change |
|------|--------|
| `internal/k8s/client.go` | +11 interface methods + implementations |
| `internal/handler/mock_client_test.go` | +11 function fields + methods |
| `internal/handler/diagnostics.go` | +3 handler methods |
| `internal/handler/modules.go` | +3 handler methods, +unstructured import |
| `internal/handler/releases.go` | +2 handler methods, +encoding/json import |
| `internal/handler/nodes.go` | +4 handler methods, +pollStaticInstance helper |
| `internal/handler/config.go` | NEW: ConfigHandler with GetClusterConfiguration |
| `internal/handler/diagnostics_test.go` | +8 tests, +fmt+resource imports |
| `internal/handler/modules_test.go` | +7 tests, +fmt import |
| `internal/handler/releases_test.go` | +5 tests, +fmt import |
| `internal/handler/nodes_test.go` | +9 tests, +corev1 import |
| `internal/handler/config_test.go` | NEW: 2 tests |
| `cmd/deckhouse-mcp/main.go` | +RegisterConfigAPITools call |
| `deploy/rbac.yaml` | +P1 RBAC permissions |
| `proto/deckhouse/v1/diagnostics.proto` | +3 RPCs +8 messages |
| `proto/deckhouse/v1/modules.proto` | +3 RPCs +6 messages |
| `proto/deckhouse/v1/releases.proto` | +2 RPCs +4 messages |
| `proto/deckhouse/v1/nodes.proto` | +4 RPCs +8 messages |
| `proto/deckhouse/v1/config.proto` | Replaced stub with real RPC +2 messages |
| `proto/deckhouse/v1/*.pb.go` | Generated |
| `proto/deckhouse/v1/*.mcp.go` | Generated (config.mcp.go new) |
