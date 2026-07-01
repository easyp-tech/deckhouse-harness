<!-- generated: 2026-05-12, template: core.md -->
# Architecture — Deckhouse MCP Server

MCP server for managing Deckhouse Kubernetes Platform (CE) via AI agents. Proto-first design: protobuf definitions are the single source of truth for all MCP tools.

## 1. Overview

**Application type:** HTTP/SSE MCP server deployed as a Kubernetes Pod.  
**Pattern:** Handler pattern — generated tool interfaces + thin handler implementations over a K8s client abstraction.

```
┌─────────────────────────────────────────────────────┐
│  Transport Layer (SSE / HTTP)                        │
│  mcp.NewSSEHandler — MCP Go SDK                      │
├─────────────────────────────────────────────────────┤
│  MCP Tool Layer (generated)                          │
│  *.mcp.go — tool registration, JSON Schema, routing  │
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
| `main.go` | Entry point: in-cluster K8s auth, MCP server creation, handler registration, SSE HTTP server, graceful shutdown |

- Creates `mcp.Server`, wraps it in `mcp.NewSSEHandler`
- Registers all tool handlers via generated `pb.Register*Tools(server, handler)` — 5 registration calls
- Listens on `:8080` (overridable via `LISTEN_ADDR`)
- Shutdown on `SIGINT`/`SIGTERM` with 30s timeout

### Proto / Generated Layer — `proto/deckhouse/v1/`

| File | Description |
|------|-------------|
| `diagnostics.proto` | Block A: `DiagnosticsAPI` — 8 RPCs (cluster status, nodes, node groups, static instances, unhealthy pods, node detail, node group detail, Deckhouse logs) |
| `modules.proto` | Block B: `ModulesAPI` — 4 RPCs (list, get, enable, disable module configs) |
| `releases.proto` | Block C: `ReleasesAPI` — 3 RPCs (list, get, approve releases) |
| `nodes.proto` | Block D: `NodesAPI` — 7 RPCs (SSH creds, static instances, add/remove node, create node group, wait ready) |
| `config.proto` | Block E: `ConfigAPI` — 1 RPC (get cluster configuration) |
| `sources.proto` | Block F: `SourcesAPI` — stub (no RPCs yet) |
| `*.pb.go` | Generated protobuf types |
| `*.mcp.go` | Generated MCP tool handler interfaces + registration functions |

Proto files are the **single source of truth**. Regenerate with `task generate`.

### Handler Layer — `internal/handler/`

| File | Description |
|------|-------------|
| `diagnostics.go` | `DiagnosticsHandler` — 8 methods: `GetClusterStatus`, `ListNodes`, `ListNodeGroups`, `ListStaticInstances`, `ListUnhealthyPods`, `GetNode`, `GetNodeGroup`, `GetDeckhouseLogs` + unstructured field helpers |
| `modules.go` | `ModulesHandler` — 4 methods: `ListModuleConfigs`, `GetModuleConfig`, `EnableModule`, `DisableModule` |
| `releases.go` | `ReleasesHandler` — 3 methods: `ListDeckhouseReleases`, `GetDeckhouseRelease`, `ApproveRelease` |
| `nodes.go` | `NodesHandler` — 7 methods: `CreateSSHCredentials`, `CreateStaticInstance`, `AddWorkerNode` (composite), `DeleteStaticInstance`, `RemoveNode` (composite), `CreateNodeGroup`, `WaitNodeReady` |
| `config.go` | `ConfigHandler` — 1 method: `GetClusterConfiguration` |
| `mock_client_test.go` | `mockClient` struct — function-field test double for `k8s.Client` |
| `*_test.go` | Unit tests (70 total across 5 test files) |

Each handler struct holds a single `k8s.Client` field. Constructor: `New{Name}Handler(client k8s.Client)`.

### K8s Client Layer — `internal/k8s/`

| File | Description |
|------|-------------|
| `client.go` | `Client` interface (17 methods) + `client` struct (typed + dynamic), GVR constants |

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
│   ├── handler/                # Tool handler implementations (5 handler files)
│   │   ├── diagnostics.go      # 8 methods
│   │   ├── modules.go          # 4 methods
│   │   ├── releases.go         # 3 methods
│   │   ├── nodes.go            # 7 methods
│   │   ├── config.go           # 1 method
│   │   └── *_test.go           # 70 unit tests
│   └── k8s/
│       └── client.go           # K8s abstraction (17-method interface)
├── proto/
│   └── deckhouse/v1/
│       ├── *.proto             # Source of truth (6 service files)
│       ├── *.pb.go             # Generated: types
│       └── *.mcp.go            # Generated: MCP bindings
├── deploy/
│   ├── deployment.yaml         # K8s Deployment (d8-system)
│   ├── rbac.yaml               # ServiceAccount + ClusterRole (P0+P1 perms)
│   └── service.yaml            # K8s Service
├── tests/integration/          # Kind-based integration tests
├── Dockerfile                  # Multi-stage builder
├── Taskfile.yml                # Build/test tasks
└── easyp.yaml                  # Proto codegen config
```

## 4. Key Design Decisions

1. **Proto-first MCP tools** — All MCP tools are defined in `.proto` files and code-generated via `protoc-gen-mcp`. Handlers implement generated interfaces. This enforces schema consistency and eliminates manual JSON Schema maintenance.

2. **K8s Client interface** — All Kubernetes API calls go through `internal/k8s.Client` (17 methods). Handlers never import `client-go` directly. This makes unit testing trivial (function-field mock) and decouples transport from infrastructure.

3. **Dynamic client for CRDs** — Deckhouse CRDs (`NodeGroup`, `StaticInstance`, etc.) are accessed via `dynamic.Interface` with `unstructured.Unstructured`. No code generation for CRD types — the schema evolves independently.

4. **Composite handlers** — Two composite tools orchestrate multi-step K8s operations:
   - `AddWorkerNode`: `CreateSSHCredentials` → `CreateStaticInstance` → polling until ready
   - `RemoveNode`: cordon → evict pods → `DeleteStaticInstance`
   On step failure, reports what was already created/done.

5. **Secrets encoded in handler** — SSH private keys and sudo passwords are base64-encoded inside the handler before writing to K8s Secrets. Clients always send plain text.

6. **In-cluster auth only** — `rest.InClusterConfig()` is the only auth path. No kubeconfig file support — the server runs as a Pod with a ServiceAccount.

7. **No middleware / framework** — Pure `net/http` + MCP SDK's `SSEHandler`. No router, no auth middleware (handled by K8s RBAC at the SA level).

8. **Idempotent write tools** — `EnableModule`, `DisableModule`, and `ApproveRelease` are safe to call repeatedly. They return previous state to indicate whether a change was made.

## 5. Data Flow

Typical read tool request (`ListNodes`):

```
MCP Client (AI agent)
  → HTTP POST /sse  (SSE connection)
    → mcp.SSEHandler (mcp-sdk)
      → mcp.Server.CallTool("deckhouse_ListNodes", {...})
        → diagnostics.mcp.go: listNodesTool.Handler(ctx, req)
          → DiagnosticsHandler.ListNodes(ctx, *ListNodesRequest)
            → k8s.Client.ListNodes(ctx)
              → kubernetes.CoreV1().Nodes().List(...)
                ← []corev1.Node
            ← convert to *pb.ListNodesResponse
          ← ProtoJSON-encode response
        ← MCP tool result
      ← SSE event to client
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
  → CordonNode(ctx, name)              [step 1: mark unschedulable]
  → ListPods(ctx, "")                  [step 2: find pods on node]
  → DeletePod(ctx, ns, name)           [step 3: evict each non-DS pod]
  → DeleteStaticInstance(ctx, name)     [step 4: remove SI]
  ← *RemoveNodeResponse
```
