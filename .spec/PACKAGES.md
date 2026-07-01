<!-- generated: 2026-05-12, template: core.md -->
# Packages Reference — Deckhouse MCP Server

## Application Layer

### `cmd/deckhouse-harness`

**Entry point and wiring.** Creates all dependencies and starts the HTTP server.

| File | Description |
|------|-------------|
| `main.go` | In-cluster K8s config, MCP server, handler registration, SSE HTTP server, graceful shutdown |

Key calls:
- `k8s.New(cfg)` — builds the K8s client
- `pb.Register*Tools(server, handler)` — registers MCP tools (5 calls: Diagnostics, Modules, Releases, Nodes, Config)
- `mcp.NewSSEHandler(...)` — wraps MCP server for HTTP/SSE transport
- `httpServer.Shutdown(ctx)` — 30s graceful shutdown

---

## Handler Layer

### `internal/handler`

**MCP tool handler implementations.** Each file implements a generated `*APIToolHandler` interface.

| File | Description |
|------|-------------|
| `diagnostics.go` | `DiagnosticsHandler` — `GetClusterStatus`, `ListNodes`, `ListNodeGroups`, `ListStaticInstances`, `ListUnhealthyPods`, `GetNode`, `GetNodeGroup`, `GetDeckhouseLogs` + helpers |
| `modules.go` | `ModulesHandler` — `ListModuleConfigs`, `GetModuleConfig`, `EnableModule`, `DisableModule` |
| `releases.go` | `ReleasesHandler` — `ListDeckhouseReleases`, `GetDeckhouseRelease`, `ApproveRelease` |
| `nodes.go` | `NodesHandler` — `CreateSSHCredentials`, `CreateStaticInstance`, `AddWorkerNode` (composite, polls), `DeleteStaticInstance`, `RemoveNode` (composite: cordon+drain+delete), `CreateNodeGroup`, `WaitNodeReady` (polling) |
| `config.go` | `ConfigHandler` — `GetClusterConfiguration` |
| `mock_client_test.go` | `mockClient` — function-field test double for `k8s.Client` (17 function fields) |
| `diagnostics_test.go` | Unit tests for `DiagnosticsHandler` |
| `modules_test.go` | Unit tests for `ModulesHandler` |
| `releases_test.go` | Unit tests for `ReleasesHandler` |
| `nodes_test.go` | Unit tests for `NodesHandler` |
| `config_test.go` | Unit tests for `ConfigHandler` |
| `errors_test.go` | Unit tests for error cases |

**Total: 70 unit tests** across test files.

Key patterns:
- Implements generated interface (e.g., `pb.DiagnosticsAPIToolHandler`)
- All K8s calls through `k8s.Client` field
- Helpers for unstructured fields: `unstructuredNestedString`, `unstructuredNestedInt64`, etc.
- `pollInterval = 30s`, `defaultTimeoutSeconds = 900`

---

## K8s Client Layer

### `internal/k8s`

**Kubernetes API abstraction.** Isolates all `client-go` usage behind an interface.

| File | Description |
|------|-------------|
| `client.go` | `Client` interface (17 methods), `client` struct, GVR constants, all K8s method implementations |

`Client` interface — current methods:
```go
// Core resources (typed)
ListNodes(ctx) ([]corev1.Node, error)
GetNode(ctx, name) (*corev1.Node, error)
CordonNode(ctx, name) error
ListPods(ctx, namespace) ([]corev1.Pod, error)
DeletePod(ctx, namespace, name) error
ListNodeEvents(ctx, nodeName) ([]corev1.Event, error)
GetPodLogs(ctx, namespace, pod, container, tail, since) (string, error)
GetSecret(ctx, namespace, name) (*corev1.Secret, error)

// Deckhouse CRDs (dynamic/unstructured)
ListNodeGroups(ctx) ([]unstructured.Unstructured, error)
GetNodeGroup(ctx, name) (*unstructured.Unstructured, error)
CreateNodeGroup(ctx, obj) (*unstructured.Unstructured, error)
ListStaticInstances(ctx) ([]unstructured.Unstructured, error)
GetStaticInstance(ctx, name) (*unstructured.Unstructured, error)
CreateStaticInstance(ctx, obj) (*unstructured.Unstructured, error)
DeleteStaticInstance(ctx, name) error
ListModuleConfigs(ctx) ([]unstructured.Unstructured, error)
GetModuleConfig(ctx, name) (*unstructured.Unstructured, error)
UpdateModuleConfig(ctx, obj) (*unstructured.Unstructured, error)
ListDeckhouseReleases(ctx) ([]unstructured.Unstructured, error)
GetDeckhouseRelease(ctx, name) (*unstructured.Unstructured, error)
PatchDeckhouseRelease(ctx, name, patch) (*unstructured.Unstructured, error)
CreateSSHCredentials(ctx, obj) (*unstructured.Unstructured, error)
```

GVR constants:
```go
NodeGroupGVR        // deckhouse.io/v1/nodegroups
StaticInstanceGVR   // deckhouse.io/v1alpha2/staticinstances
SSHCredentialsGVR   // deckhouse.io/v1alpha2/sshcredentials
ModuleConfigGVR     // deckhouse.io/v1alpha1/moduleconfigs
DeckhouseReleaseGVR // deckhouse.io/v1alpha1/deckhouserelease
```

---

## Proto / Generated Layer

### `proto/deckhouse/v1`

**Source of truth for MCP tools.** Do not manually edit `*.pb.go` or `*.mcp.go`.

| File | Description |
|------|-------------|
| `diagnostics.proto` | Block A: `DiagnosticsAPI` service — 8 RPCs |
| `diagnostics.pb.go` | Generated protobuf types for diagnostics |
| `diagnostics.mcp.go` | Generated: `DiagnosticsAPIToolHandler` interface + `RegisterDiagnosticsAPITools()` |
| `modules.proto` | Block B: `ModulesAPI` — 4 RPCs |
| `modules.pb.go` | Generated types |
| `modules.mcp.go` | Generated: `ModulesAPIToolHandler` + registration |
| `releases.proto` | Block C: `ReleasesAPI` — 3 RPCs |
| `releases.pb.go` | Generated types |
| `releases.mcp.go` | Generated: `ReleasesAPIToolHandler` + registration |
| `nodes.proto` | Block D: `NodesAPI` — 7 RPCs |
| `nodes.pb.go` | Generated types |
| `nodes.mcp.go` | Generated: `NodesAPIToolHandler` + registration |
| `config.proto` | Block E: `ConfigAPI` — 1 RPC |
| `config.pb.go` | Generated types |
| `config.mcp.go` | Generated: `ConfigAPIToolHandler` + registration |
| `sources.proto` | Block F: `SourcesAPI` — stub (no RPCs yet) |
| `sources.pb.go` | Generated (empty service) |

Regenerate everything: `task generate` (runs `easyp mod download && easyp generate`).

---

## Deploy / Integration

### `deploy/`

| File | Description |
|------|-------------|
| `deployment.yaml` | K8s Deployment — 1 replica, `d8-system` namespace, resource limits |
| `rbac.yaml` | ServiceAccount + ClusterRole + ClusterRoleBinding (P0+P1 permissions) |
| `service.yaml` | K8s Service (ClusterIP, port 8080) |

### `tests/integration/`

| File | Description |
|------|-------------|
| `setup.sh` | Creates Kind cluster, loads CRDs, applies fixtures |
| `test.sh` | Sends MCP tool calls, validates responses |
| `teardown.sh` | Deletes Kind cluster |
| `crds.yaml` | Deckhouse CRD definitions for local testing |
| `fixtures.yaml` | Sample K8s resources (nodes, nodegroups, etc.) |
