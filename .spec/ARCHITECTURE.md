<!-- generated: 2026-07-07, template: core.md -->
# Architecture — Deckhouse Harness

MCP server for managing Deckhouse Kubernetes Platform (CE) via AI agents. Proto-first design: protobuf definitions are the single source of truth for all MCP tools.

## 1. Overview

**Application type:** stdio MCP server — a client launches the binary and speaks newline-delimited JSON-RPC over stdin/stdout. Exposes Tools, Resources, and Prompts.  
**Pattern:** Handler pattern — generated interfaces + thin handler implementations over a K8s client abstraction.

```
┌─────────────────────────────────────────────────────┐
│  Transport Layer (stdio)                             │
│  mcpruntime.ServeStdio(ctx, server)                  │
├─────────────────────────────────────────────────────┤
│  MCP Layer (generated) — Tools · Resources · Prompts │
│  *.mcp.go — registration, JSON Schema, routing       │
├─────────────────────────────────────────────────────┤
│  Handler Layer                                       │
│  internal/handler/*.go — business logic              │
├─────────────────────────────────────────────────────┤
│  K8s Client Abstraction                              │
│  internal/k8s/client.go — Client interface           │
├─────────────────────────────────────────────────────┤
│  Kubernetes API (in-cluster)                         │
│  Typed client (core resources) + Dynamic (CRDs)      │
└─────────────────────────────────────────────────────┘
```

## 2. Component Deep Dive

### Transport Layer — `cmd/deckhouse-harness/main.go`

| File | Description |
|------|-------------|
| `main.go` | Entry point (`urfave/cli/v3`): K8s auth, MCP server creation, tool/resource/prompt registration, stdio serve, graceful shutdown |

- K8s auth: `loadKubeConfig()` tries `rest.InClusterConfig()` first, then falls back to `KUBECONFIG` / `~/.kube/config`
- Creates `mcpruntime.NewServer(name, version)` (self-contained runtime; no `modelcontextprotocol/go-sdk`)
- Registers the 6 tool services (`pb.Register*Tools`), the resources (`RegisterFile_..._Resources`), and the prompts (`RegisterFile_..._Prompts`)
- Serves over stdio via `mcpruntime.ServeStdio(ctx, server)`; blocks until stdin closes or ctx is cancelled
- Shutdown on `SIGINT`/`SIGTERM` via context cancellation

### Proto / Generated Layer — `proto/deckhouse/v1/`

| File | Description |
|------|-------------|
| `diagnostics.proto` | Block A: `DiagnosticsAPI` — 11 RPCs (`GetClusterStatus`, `ListNodes`, `ListNodeGroups`, `ListStaticInstances`, `ListUnhealthyPods`, `GetNode`, `GetNodeGroup`, `GetDeckhouseLogs`, `GetNodeEvents`, `GetStaticInstance`, `GetPodLogs`) |
| `modules.proto` | Block B: `ModulesAPI` — 7 RPCs (`ListModuleConfigs`, `GetModuleConfig`, `EnableModule`, `DisableModule`, `ListModules`, `UpdateModuleSettings`, `SetModuleMaintenance`) |
| `releases.proto` | Block C: `ReleasesAPI` — 3 RPCs (`ListDeckhouseReleases`, `GetDeckhouseRelease`, `ApproveRelease`) |
| `nodes.proto` | Block D: `NodesAPI` — 13 RPCs (`CreateSSHCredentials`, `DeleteSSHCredentials`, `CreateStaticInstance`, `DeleteStaticInstance`, `AddWorkerNode`, `RemoveNode`, `CreateNodeGroup`, `DeleteNodeGroup`, `WaitNodeReady`, `CordonNode`, `UncordonNode`, `DrainNode`, `CreateNodeGroupConfiguration`) |
| `config.proto` | Block E: `ConfigAPI` — 3 RPCs (`GetClusterConfiguration`, `GetStaticClusterConfiguration`, `UpdateKubernetesVersion`) |
| `sources.proto` | Block F: `SourcesAPI` — 6 RPCs (`ListModuleSources`, `CreateModuleSource`, `DeleteModuleSource`, `ListModuleUpdatePolicies`, `CreateModuleUpdatePolicy`, `ListModuleReleases`) |
| `*.pb.go` | Generated protobuf types |
| `*.mcp.go` | Generated MCP tool handler interfaces + registration functions |

Proto files are the **single source of truth**. Regenerate with `task generate`.

### Handler Layer — `internal/handler/`

| File | Description |
|------|-------------|
| `diagnostics.go` | `DiagnosticsHandler` — 11 tools: `GetClusterStatus`, `ListNodes`, `ListNodeGroups`, `ListStaticInstances`, `ListUnhealthyPods`, `GetNode`, `GetNodeGroup`, `GetDeckhouseLogs`, `GetNodeEvents`, `GetStaticInstance`, `GetPodLogs` + unstructured field helpers |
| `modules.go` | `ModulesHandler` — 7 tools: `ListModuleConfigs`, `GetModuleConfig`, `EnableModule`, `DisableModule`, `ListModules`, `UpdateModuleSettings`, `SetModuleMaintenance` |
| `releases.go` | `ReleasesHandler` — 3 tools: `ListDeckhouseReleases`, `GetDeckhouseRelease`, `ApproveRelease` |
| `nodes.go` | `NodesHandler` — 13 tools: `CreateSSHCredentials`, `DeleteSSHCredentials`, `CreateStaticInstance`, `DeleteStaticInstance`, `AddWorkerNode` (composite), `RemoveNode` (composite), `CreateNodeGroup`, `DeleteNodeGroup`, `WaitNodeReady`, `CordonNode`, `UncordonNode`, `DrainNode` (composite), `CreateNodeGroupConfiguration` |
| `config.go` | `ConfigHandler` — 3 tools: `GetClusterConfiguration`, `GetStaticClusterConfiguration`, `UpdateKubernetesVersion` |
| `sources.go` | `SourcesHandler` — 6 tools: `ListModuleSources`, `CreateModuleSource`, `DeleteModuleSource`, `ListModuleUpdatePolicies`, `CreateModuleUpdatePolicy`, `ListModuleReleases` |
| `resources.go` | `ResourcesHandler` — MCP resources (5 static + 2 templated), delegates to the tool handlers |
| `prompts.go` | `PromptsHandler` — MCP prompts (5 diagnostic playbooks) |
| `mock_client_test.go` | `mockClient` struct — function-field test double for `k8s.Client` |
| `*_test.go` | Unit tests (156 total; polling tests use a real 30s clock, ~3 min runtime) |

Each handler struct holds a single `k8s.Client` field. Constructor: `New{Name}Handler(client k8s.Client)`.

### K8s Client Layer — `internal/k8s/`

| File | Description |
|------|-------------|
| `client.go` | `Client` interface (36 methods: 11 core + 25 CRD) + `client` struct (typed + dynamic), GVR constants |

Two underlying clients:
- **Typed** (`kubernetes.Interface`) — core resources: `nodes`, `pods`, `events`, `secrets`, `pods/log`
- **Dynamic** (`dynamic.Interface`) — Deckhouse CRDs via `unstructured.Unstructured`

## 3. Directory Structure

```
deckhouse-harness/
├── cmd/
│   └── deckhouse-harness/
│       └── main.go             # Entrypoint
├── internal/
│   ├── handler/                # Handler implementations (tools + resources + prompts)
│   │   ├── diagnostics.go      # 11 tools
│   │   ├── modules.go          # 7 tools
│   │   ├── releases.go         # 3 tools
│   │   ├── nodes.go            # 13 tools
│   │   ├── config.go           # 3 tools
│   │   ├── sources.go          # 6 tools
│   │   ├── resources.go        # MCP resources (delegate to tool handlers)
│   │   ├── prompts.go          # MCP prompts (diagnostic playbooks)
│   │   └── *_test.go           # 156 unit tests
│   └── k8s/
│       └── client.go           # K8s abstraction (36-method interface)
├── proto/
│   └── deckhouse/v1/
│       ├── *.proto             # Source of truth (6 tool services + resources + prompts)
│       ├── *.pb.go             # Generated: types
│       └── *.mcp.go            # Generated: MCP bindings (tools, resources, prompts)
├── deploy/
│   ├── rbac.yaml               # ServiceAccount + ClusterRole/Binding (all 43 tools)
│   └── README.md               # stdio run model (no HTTP service)
├── tests/integration/          # Kind-based integration tests
├── Dockerfile                  # Multi-stage builder
├── Taskfile.yml                # Build/test tasks
└── easyp.yaml                  # Proto codegen config
```

## 4. Key Design Decisions

1. **Proto-first MCP tools** — All MCP tools are defined in `.proto` files and code-generated via `protoc-gen-mcp`. Handlers implement generated interfaces. This enforces schema consistency and eliminates manual JSON Schema maintenance.

2. **K8s Client interface** — All Kubernetes API calls go through `internal/k8s.Client` (36 methods: 11 core + 25 CRD). Handlers never import `client-go` directly. This makes unit testing trivial (function-field mock) and decouples transport from infrastructure.

3. **Dynamic client for CRDs** — Deckhouse CRDs (`NodeGroup`, `StaticInstance`, etc.) are accessed via `dynamic.Interface` with `unstructured.Unstructured`. No code generation for CRD types — the schema evolves independently.

4. **Composite handlers** — Three composite tools orchestrate multi-step K8s operations:
   - `AddWorkerNode`: `CreateSSHCredentials` → `CreateStaticInstance` → poll until `Running`
   - `RemoveNode`: drain node → `DeleteStaticInstance`
   - `DrainNode`: cordon + PDB-aware eviction loop
   On step failure, reports what was already created/done.

5. **Secrets encoded in handler** — SSH private keys and sudo passwords are base64-encoded inside the handler before writing to K8s Secrets. Clients always send plain text.

6. **Flexible K8s auth** — `loadKubeConfig()` tries `rest.InClusterConfig()` first (Pod + ServiceAccount), then falls back to `KUBECONFIG` / `~/.kube/config`. This supports both in-cluster deployment and local development against a remote cluster.

7. **stdio, no middleware / framework** — the server speaks stdio only (`mcpruntime.ServeStdio`). No HTTP listener, no router, no auth middleware (in-cluster access handled by K8s RBAC at the SA level). For HTTP fronting, wrap `Server.HandleRaw` externally.

8. **Idempotent write tools** — `EnableModule`, `DisableModule`, and `ApproveRelease` are safe to call repeatedly. They return previous state to indicate whether a change was made.

## 5. Data Flow

Typical read tool request (`ListNodes`) over stdio (newline-delimited JSON-RPC on stdin/stdout):

```
MCP Client (AI agent)
  → stdin JSON-RPC line
    → mcpruntime.ServeStdio → Server.HandleRaw
      → tools/call "deckhouse_ListNodes" {...}
        → diagnostics.mcp.go: listNodesTool.Handler(ctx, req)
          → DiagnosticsHandler.ListNodes(ctx, *ListNodesRequest)
            → k8s.Client.ListNodes(ctx)
              → kubernetes.CoreV1().Nodes().List(...)
                ← []corev1.Node
            ← convert to *pb.ListNodesResponse
          ← ProtoJSON-encode response
        ← MCP tool result
      ← stdout JSON-RPC line to client
```

Write tool with polling (`AddWorkerNode`):

```
NodesHandler.AddWorkerNode(ctx, req)
  → CreateSSHCredentials(ctx, sshObj)   [step 1]
  → CreateStaticInstance(ctx, siObj)    [step 2]
  → loop every 30s (max 15 min):
      GetStaticInstance(ctx, name)
      check .status.currentStatus.phase == "Running"
      if timeout → return {timedOut: true, lastStatus: ...}
  ← *AddWorkerNodeResponse
```

Composite write tool (`RemoveNode`):

```
NodesHandler.RemoveNode(ctx, req)
  → DrainNode(ctx, name)               [step 1: cordon + PDB-aware eviction loop]
      CordonNode(ctx, name)              (mark unschedulable)
      ListPods(ctx, "") + EvictPod(...)  (evict each non-DS pod, respecting PDBs)
  → DeleteStaticInstance(ctx, name)     [step 2: remove SI]
  ← *RemoveNodeResponse
```
