# Deckhouse Harness

MCP server for managing [Deckhouse Kubernetes Platform](https://deckhouse.ru/docs) (Community Edition).

Transport: **stdio** — newline-delimited JSON-RPC on stdin/stdout. A client
launches the binary and speaks MCP over its standard streams.

Exposes three MCP primitives: **Tools** (43), **Resources** (read-only cluster
context by URI), and **Prompts** (parameterized diagnostic playbooks).

Authenticates to Kubernetes using in-cluster config (when running inside a Pod) or `~/.kube/config` / `KUBECONFIG` (for local execution).

## Features

**Diagnostics** (read-only, 11 tools)
- `deckhouse_GetClusterStatus` — aggregated cluster health: nodes, modules, releases, unhealthy pods
- `deckhouse_ListNodes` — cluster nodes with filtering by group, status, role
- `deckhouse_ListNodeGroups` — NodeGroup resources with status and conditions
- `deckhouse_ListStaticInstances` — StaticInstance resources with filtering by group and phase
- `deckhouse_ListUnhealthyPods` — pods not in Running/Succeeded state
- `deckhouse_GetNode` — detailed node info with conditions, capacity, events
- `deckhouse_GetNodeGroup` — full NodeGroup spec with member node names
- `deckhouse_GetDeckhouseLogs` — Deckhouse controller pod logs with grep/tail/since
- `deckhouse_GetNodeEvents` — Kubernetes Events for a specific node
- `deckhouse_GetStaticInstance` — detailed StaticInstance info
- `deckhouse_GetPodLogs` — logs for a specific pod and container

**Modules** (7 tools)
- `deckhouse_ListModuleConfigs` — ModuleConfig resources with enabled/disabled filter
- `deckhouse_GetModuleConfig` — full spec and status of a single ModuleConfig
- `deckhouse_EnableModule` / `deckhouse_DisableModule` — toggle module enabled state
- `deckhouse_ListModules` — runtime Module resources
- `deckhouse_UpdateModuleSettings` — RFC 7396 JSON Merge Patch on module settings
- `deckhouse_SetModuleMaintenance` — toggle module maintenance mode

**Releases** (3 tools)
- `deckhouse_ListDeckhouseReleases` — DeckhouseRelease resources with phase filter
- `deckhouse_GetDeckhouseRelease` — full release details with requirements
- `deckhouse_ApproveRelease` — approve a pending release

**Nodes** (13 tools, write)
- `deckhouse_CreateSSHCredentials` / `deckhouse_DeleteSSHCredentials`
- `deckhouse_CreateStaticInstance` / `deckhouse_DeleteStaticInstance`
- `deckhouse_AddWorkerNode` — composite: SSHCredentials → StaticInstance → wait for Running
- `deckhouse_RemoveNode` — composite: drain → delete StaticInstance
- `deckhouse_CreateNodeGroup` / `deckhouse_DeleteNodeGroup`
- `deckhouse_WaitNodeReady` — poll StaticInstance until Running or timeout
- `deckhouse_CordonNode` / `deckhouse_UncordonNode`
- `deckhouse_DrainNode` — cordon + eviction loop with PDB awareness
- `deckhouse_CreateNodeGroupConfiguration` — bash script bound to NodeGroups

**Config** (3 tools)
- `deckhouse_GetClusterConfiguration` — read ClusterConfiguration YAML
- `deckhouse_GetStaticClusterConfiguration` — read StaticClusterConfiguration YAML
- `deckhouse_UpdateKubernetesVersion` — patch kubernetesVersion with retry-on-conflict

**Sources** (6 tools)
- `deckhouse_ListModuleSources` / `deckhouse_CreateModuleSource` / `deckhouse_DeleteModuleSource`
- `deckhouse_ListModuleUpdatePolicies` / `deckhouse_CreateModuleUpdatePolicy`
- `deckhouse_ListModuleReleases` — module releases with phase filter

**Resources** (read-only context by URI)
- `deckhouse://cluster/status` — aggregated cluster health
- `deckhouse://cluster/configuration` — ClusterConfiguration
- `deckhouse://nodes` — all nodes · `deckhouse://nodes/{name}` — one node
- `deckhouse://modules` — all modules · `deckhouse://modules/{name}` — one module's config
- `deckhouse://releases` — all DeckhouseReleases

**Prompts** (parameterized playbooks)
- `diagnose_cluster_health` — full cluster-health assessment
- `triage_unhealthy_pods` — root-cause failing pods
- `investigate_node` — deep-dive a single node
- `prepare_deckhouse_upgrade` — pre-upgrade readiness review
- `add_worker_node` — step-by-step worker onboarding

## Tech Stack

- **Go 1.26** with [protoc-gen-mcp](https://github.com/easyp-tech/protoc-gen-mcp) `mcpruntime` (self-contained stdio MCP runtime)
- **Protobuf + [protoc-gen-mcp](https://github.com/easyp-tech/protoc-gen-mcp)** — proto-first Tools, Resources, and Prompts generation
- **[EasyP](https://github.com/easyp-tech/easyp)** — proto linting, codegen, dependency management
- **client-go** — typed client for core resources, dynamic client for Deckhouse CRDs

## Prerequisites

- Go 1.26+
- [EasyP](https://github.com/easyp-tech/easyp) (`brew install easyp-tech/tap/easyp`)
- [go-task](https://taskfile.dev) (`brew install go-task`)
- A Kubernetes cluster with Deckhouse CE installed
- `~/.kube/config` or `KUBECONFIG` env pointing to the cluster

## Quick Start

```bash
# Generate protobuf code
task generate

# Build
go build -o deckhouse-harness ./cmd/deckhouse-harness

# Run tests
task test

# Run the server (connects to cluster via kubeconfig)
./deckhouse-harness
```

## Transport

The server speaks **stdio** only — newline-delimited JSON-RPC on stdin/stdout.
Used by MCP clients like Claude Desktop, Cursor, and Claude Code, which launch the
binary as a subprocess. Logs go to stderr (stdout is reserved for the protocol).

```bash
./deckhouse-harness    # reads JSON-RPC from stdin, writes responses to stdout
```

There is no HTTP/SSE listener. For in-cluster use, run the binary in a Pod with the
provided ServiceAccount (see [`deploy/`](deploy/README.md)).

## Connecting an MCP Client

Configure your MCP client to launch the binary; it communicates over stdio.

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS):

```json
{
  "mcpServers": {
    "deckhouse": {
      "command": "/path/to/deckhouse-harness",
      "env": {
        "KUBECONFIG": "/Users/you/.kube/config",
        "LOG_LEVEL": "INFO"
      }
    }
  }
}
```

### Cursor / VS Code MCP

```json
{
  "mcpServers": {
    "deckhouse": {
      "command": "/path/to/deckhouse-harness"
    }
  }
}
```

### Manual (pipe JSON-RPC)

```bash
echo '{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./deckhouse-harness
```

## Using Resources & Prompts

After `initialize`, the server advertises all three capabilities (`tools`,
`resources`, `prompts`). Beyond the 43 tools:

**Resources** — read-only cluster context by URI. Discover with `resources/list`
and `resources/templates/list`, then read:

```jsonc
// resources/read — static
{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"deckhouse://cluster/status"}}

// resources/read — templated (one node by name)
{"jsonrpc":"2.0","id":2,"method":"resources/read","params":{"uri":"deckhouse://nodes/worker-1"}}
```

The response body is ProtoJSON in `contents[0].text`. Note: the per-instance
enumeration in `resources/list` (e.g. `deckhouse://nodes/<name>`) is a snapshot
taken at server startup; any URI is always readable live via the template.

**Prompts** — parameterized diagnostic playbooks. Discover with `prompts/list`,
then fetch with arguments:

```jsonc
// prompts/get — required and optional arguments
{"jsonrpc":"2.0","id":3,"method":"prompts/get","params":{"name":"investigate_node","arguments":{"name":"worker-1"}}}
{"jsonrpc":"2.0","id":4,"method":"prompts/get","params":{"name":"diagnose_cluster_health","arguments":{"focus":"modules"}}}
```

The response returns a `user` message whose text is a step-by-step playbook that
orchestrates the relevant tools and resources.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KUBECONFIG` | `~/.kube/config` | Path to kubeconfig file (or in-cluster config when running inside a Pod) |
| `LOG_LEVEL` | `INFO` | Log verbosity: `DEBUG`, `INFO`, `WARN`, `ERROR` |
| `LOG_OUTPUT` | `stderr` | Log destination: `stderr`, `file`, `discard` |
| `LOG_FILE` | — | Log file path (required when `LOG_OUTPUT=file`) |

stdout is reserved for the MCP protocol; logs never go there.

## Development

### Available Tasks

```
task generate        # easyp mod download + easyp generate
task lint            # easyp lint
task build           # go build -o deckhouse-harness ./cmd/deckhouse-harness
task test            # go test ./...
task docker:build    # build Docker image (deckhouse-harness:local)
task docker:load     # load Docker image into Kind cluster
task integration     # full integration test cycle (requires Kind)
```

### Proto-First Workflow

All MCP primitives are defined in `.proto` files under `proto/deckhouse/v1/`:

| File | Primitive | Purpose |
|------|-----------|---------|
| `diagnostics.proto` | `DiagnosticsAPI` tools | Read-only cluster status, nodes, pods, logs, events |
| `modules.proto` | `ModulesAPI` tools | ModuleConfig management, enable/disable, settings |
| `releases.proto` | `ReleasesAPI` tools | DeckhouseRelease listing and approval |
| `nodes.proto` | `NodesAPI` tools | Node provisioning, drain/cordon, NodeGroup |
| `config.proto` | `ConfigAPI` tools | Cluster configuration, Kubernetes version |
| `sources.proto` | `SourcesAPI` tools | ModuleSource, ModuleUpdatePolicy, ModuleRelease |
| `resources.proto` | resources | Read-only cluster context by URI (`(mcp.options.v1.resource)`) |
| `prompts.proto` | prompts | Parameterized diagnostic playbooks (`(mcp.options.v1.prompt)`) |

After editing `.proto` files, regenerate:

```bash
task generate
```

### Implementing Handlers

Each tool handler implements a generated `*ToolHandler` interface:

```go
type DiagnosticsAPIToolHandler interface {
    GetClusterStatus(ctx context.Context, req *emptypb.Empty) (*GetClusterStatusResponse, error)
    ListNodes(ctx context.Context, req *ListNodesRequest) (*ListNodesResponse, error)
    // ...
}
```

Resources and prompts have their own generated handler interfaces:

- **Resource** message → `Read{Msg}` (static) or `List{Msg}s` + `Read{Msg}(ctx, {param})` (templated), implemented in `internal/handler/resources.go`. The `ResourcesHandler` delegates to the tool handlers — no duplicated data-access logic.
- **Prompt** message → `{Msg}(ctx, req *Msg) ([]mcpruntime.PromptMessage, error)`, implemented in `internal/handler/prompts.go`. Each field of the prompt message becomes an argument.

Handlers live in `internal/handler/` and receive a `k8s.Client` interface for all Kubernetes operations. The generated code, the server, and the runtime types all come from `protoc-gen-mcp/mcpruntime` (no `modelcontextprotocol/go-sdk`).

### Testing

```bash
task test    # unit tests (mock k8s.Client, no cluster needed)
```

Tests use a mock `k8s.Client` with function fields — no external mock libraries.

## Project Structure

```
proto/deckhouse/v1/          # .proto files — single source of truth
├── diagnostics.proto        # DiagnosticsAPI (11 RPCs)
├── modules.proto            # ModulesAPI (7 RPCs)
├── releases.proto           # ReleasesAPI (3 RPCs)
├── nodes.proto              # NodesAPI (13 RPCs)
├── config.proto             # ConfigAPI (3 RPCs)
├── sources.proto            # SourcesAPI (6 RPCs)
├── resources.proto          # MCP Resources (read-only context by URI)
└── prompts.proto            # MCP Prompts (diagnostic playbooks)
cmd/deckhouse-harness/main.go    # stdio entrypoint (mcpruntime.ServeStdio)
internal/handler/            # Tool / resource / prompt handler implementations
internal/k8s/client.go       # Kubernetes client interface
deploy/                      # ServiceAccount + RBAC (see deploy/README.md)
Dockerfile                   # Multi-stage build (golang:1.26 → distroless)
tests/integration/           # Integration tests (Kind + Deckhouse CE, stdio)
Taskfile.yml                 # Build tasks (go-task)
easyp.yaml                   # Proto config
```

## License

[MIT](LICENSE)
