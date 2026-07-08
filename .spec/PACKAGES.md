<!-- generated: 2026-07-07, template: core.md -->
# Packages Reference — Deckhouse Harness

## Application Layer

### `cmd/deckhouse-harness`

**Entry point and wiring.** Creates all dependencies and serves MCP over stdio.

| File | Description |
|------|-------------|
| `main.go` | K8s config (in-cluster or kubeconfig), `mcpruntime` server, tool/resource/prompt registration, stdio serve, graceful shutdown |

Key calls:
- `k8s.New(cfg)` — builds the K8s client
- `mcpruntime.NewServer(name, version)` — self-contained MCP server (no `modelcontextprotocol/go-sdk`)
- `pb.Register*Tools(server, handler)` — registers MCP tools (6 calls: Diagnostics, Modules, Releases, Nodes, Config, Sources)
- `pb.RegisterFile_..._Resources(ctx, server, handler)` / `pb.RegisterFile_..._Prompts(server, handler)` — registers resources and prompts
- `mcpruntime.ServeStdio(ctx, server)` — MCP over stdin/stdout; blocks until stdin closes or ctx is cancelled

---

## Handler Layer

### `internal/handler`

**MCP handler implementations.** Tool handlers implement a generated `*APIToolHandler` interface; the resource and prompt handlers implement the generated resource/prompt interfaces.

| File | Description |
|------|-------------|
| `diagnostics.go` | `DiagnosticsHandler` (11) — `GetClusterStatus`, `ListNodes`, `ListNodeGroups`, `ListStaticInstances`, `ListUnhealthyPods`, `GetNode`, `GetNodeGroup`, `GetDeckhouseLogs`, `GetNodeEvents`, `GetStaticInstance`, `GetPodLogs` + helpers |
| `modules.go` | `ModulesHandler` (7) — `ListModuleConfigs`, `GetModuleConfig`, `EnableModule`, `DisableModule`, `ListModules`, `UpdateModuleSettings`, `SetModuleMaintenance` |
| `releases.go` | `ReleasesHandler` (3) — `ListDeckhouseReleases`, `GetDeckhouseRelease`, `ApproveRelease` |
| `nodes.go` | `NodesHandler` (13) — `CreateSSHCredentials`, `DeleteSSHCredentials`, `CreateStaticInstance`, `DeleteStaticInstance`, `AddWorkerNode` (composite, polls), `RemoveNode` (composite: cordon+drain+delete), `CreateNodeGroup`, `DeleteNodeGroup`, `WaitNodeReady` (polling), `CordonNode`, `UncordonNode`, `DrainNode` (composite), `CreateNodeGroupConfiguration` |
| `config.go` | `ConfigHandler` (3) — `GetClusterConfiguration`, `GetStaticClusterConfiguration`, `UpdateKubernetesVersion` |
| `sources.go` | `SourcesHandler` (6) — `ListModuleSources`, `CreateModuleSource`, `DeleteModuleSource`, `ListModuleUpdatePolicies`, `CreateModuleUpdatePolicy`, `ListModuleReleases` |
| `resources.go` | `ResourcesHandler` — MCP resources (5 static + 2 templated); delegates to the tool handlers above |
| `prompts.go` | `PromptsHandler` — MCP prompts (5 playbooks); returns user `TextContent` messages |
| `mock_client_test.go` | `mockClient` — function-field test double for `k8s.Client` (36 function fields) |
| `*_test.go` | Per-handler unit tests: `diagnostics_test.go`, `modules_test.go`, `releases_test.go`, `nodes_test.go`, `config_test.go`, `sources_test.go`, `resources_test.go`, `prompts_test.go`, `errors_test.go` |

Key patterns:
- Implements the generated interface (e.g., `pb.DiagnosticsAPIToolHandler`, or the resource/prompt handler interfaces)
- All K8s calls through `k8s.Client` field; resources/prompts reuse the tool handlers (no duplicated logic)
- Helpers for unstructured fields: `unstructuredNestedString`, `unstructuredNestedInt64`, etc.
- `pollInterval = 30s`, `defaultTimeoutSeconds = 900`

---

## K8s Client Layer

### `internal/k8s`

**Kubernetes API abstraction.** Isolates all `client-go` usage behind an interface.

| File | Description |
|------|-------------|
| `client.go` | `Client` interface (36 methods: 11 core + 25 CRD), `client` struct, GVR constants, all K8s method implementations |

`Client` interface — current methods:
```go
// Core resources (typed) — 11
ListNodes(ctx) ([]corev1.Node, error)
GetNode(ctx, name) (*corev1.Node, error)
CordonNode(ctx, name) error
UncordonNode(ctx, name) error
ListPods(ctx, namespace) ([]corev1.Pod, error)
DeletePod(ctx, namespace, name) error
EvictPod(ctx, namespace, name) error
ListNodeEvents(ctx, nodeName) ([]corev1.Event, error)
GetPodLogs(ctx, namespace, pod, container, tail, since) (string, error)
GetSecret(ctx, namespace, name) (*corev1.Secret, error)
UpdateSecret(ctx, secret) (*corev1.Secret, error)

// Deckhouse CRDs (dynamic/unstructured) — 25
ListNodeGroups(ctx) ([]unstructured.Unstructured, error)
GetNodeGroup(ctx, name) (*unstructured.Unstructured, error)
CreateNodeGroup(ctx, obj) (*unstructured.Unstructured, error)
DeleteNodeGroup(ctx, name) error
ListStaticInstances(ctx) ([]unstructured.Unstructured, error)
GetStaticInstance(ctx, name) (*unstructured.Unstructured, error)
CreateStaticInstance(ctx, obj) (*unstructured.Unstructured, error)
DeleteStaticInstance(ctx, name) error
ListModuleConfigs(ctx) ([]unstructured.Unstructured, error)
GetModuleConfig(ctx, name) (*unstructured.Unstructured, error)
UpdateModuleConfig(ctx, obj) (*unstructured.Unstructured, error)
PatchModuleConfig(ctx, name, patch) (*unstructured.Unstructured, error)
ListDeckhouseReleases(ctx) ([]unstructured.Unstructured, error)
GetDeckhouseRelease(ctx, name) (*unstructured.Unstructured, error)
PatchDeckhouseRelease(ctx, name, patch) (*unstructured.Unstructured, error)
CreateSSHCredentials(ctx, obj) (*unstructured.Unstructured, error)
DeleteSSHCredentials(ctx, name) error
ListModules(ctx) ([]unstructured.Unstructured, error)
ListModuleSources(ctx) ([]unstructured.Unstructured, error)
CreateModuleSource(ctx, obj) (*unstructured.Unstructured, error)
DeleteModuleSource(ctx, name) error
ListModuleUpdatePolicies(ctx) ([]unstructured.Unstructured, error)
CreateModuleUpdatePolicy(ctx, obj) (*unstructured.Unstructured, error)
ListModuleReleases(ctx) ([]unstructured.Unstructured, error)
CreateNodeGroupConfiguration(ctx, obj) (*unstructured.Unstructured, error)
```

Note: `ListNodeGroups`/`ListStaticInstances` return an actionable error
`"CRD deckhouse.io/v1/nodegroups not registered (is node-manager module enabled?)"`
when the CRD is absent (e.g. the node-manager module is disabled).

GVR constants (10 Deckhouse CRDs):
```go
NodeGroupGVR              // deckhouse.io/v1/nodegroups
StaticInstanceGVR         // deckhouse.io/v1alpha2/staticinstances
SSHCredentialsGVR         // deckhouse.io/v1alpha2/sshcredentials
ModuleConfigGVR           // deckhouse.io/v1alpha1/moduleconfigs
DeckhouseReleaseGVR       // deckhouse.io/v1alpha1/deckhouserelease
ModuleGVR                 // deckhouse.io/v1alpha1/modules
ModuleSourceGVR           // deckhouse.io/v1alpha1/modulesources
ModuleUpdatePolicyGVR     // deckhouse.io/v1alpha1/moduleupdatepolicies
ModuleReleaseGVR          // deckhouse.io/v1alpha1/modulereleases
NodeGroupConfigurationGVR // deckhouse.io/v1alpha1/nodegroupconfigurations
```

---

## Proto / Generated Layer

### `proto/deckhouse/v1`

**Source of truth for MCP tools.** Do not manually edit `*.pb.go` or `*.mcp.go`.

| File | Description |
|------|-------------|
| `diagnostics.proto` | Block A: `DiagnosticsAPI` service — 11 RPCs |
| `diagnostics.pb.go` | Generated protobuf types for diagnostics |
| `diagnostics.mcp.go` | Generated: `DiagnosticsAPIToolHandler` interface + `RegisterDiagnosticsAPITools()` |
| `modules.proto` | Block B: `ModulesAPI` — 7 RPCs |
| `modules.pb.go` | Generated types |
| `modules.mcp.go` | Generated: `ModulesAPIToolHandler` + registration |
| `releases.proto` | Block C: `ReleasesAPI` — 3 RPCs |
| `releases.pb.go` | Generated types |
| `releases.mcp.go` | Generated: `ReleasesAPIToolHandler` + registration |
| `nodes.proto` | Block D: `NodesAPI` — 13 RPCs |
| `nodes.pb.go` | Generated types |
| `nodes.mcp.go` | Generated: `NodesAPIToolHandler` + registration |
| `config.proto` | Block E: `ConfigAPI` — 3 RPCs |
| `config.pb.go` | Generated types |
| `config.mcp.go` | Generated: `ConfigAPIToolHandler` + registration |
| `sources.proto` | Block F: `SourcesAPI` — 6 RPCs |
| `sources.pb.go` | Generated types |
| `sources.mcp.go` | Generated: `SourcesAPIToolHandler` + registration |

Regenerate everything: `task generate` (runs `easyp mod download && easyp generate`).

---

## Deploy / Integration

### `deploy/`

| File | Description |
|------|-------------|
| `rbac.yaml` | ServiceAccount + ClusterRole + ClusterRoleBinding (all P0–P3 permissions) |
| `README.md` | stdio deployment model (no HTTP Deployment/Service) |

### `tests/integration/`

| File | Description |
|------|-------------|
| `setup.sh` | Creates Kind cluster, loads CRDs, applies fixtures |
| `test.sh` | Sends MCP tool calls, validates responses |
| `teardown.sh` | Deletes Kind cluster |
| `crds.yaml` | Deckhouse CRD definitions for local testing |
| `fixtures.yaml` | Sample K8s resources (nodes, nodegroups, etc.) |
