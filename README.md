# Deckhouse MCP Server

MCP server for managing [Deckhouse Kubernetes Platform](https://deckhouse.ru/docs) (Community Edition).

Deployed as a Pod in `d8-system` namespace, authenticated via ServiceAccount + RBAC.
Communicates over SSE (HTTP) transport using the [Model Context Protocol](https://spec.modelcontextprotocol.io).

## Features

**Diagnostics** (read-only)
- `deckhouse_GetClusterStatus` — aggregated cluster health: nodes, modules, releases, unhealthy pods
- `deckhouse_ListNodes` — cluster nodes with filtering by group, status, role
- `deckhouse_ListNodeGroups` — NodeGroup resources with status and conditions
- `deckhouse_ListStaticInstances` — StaticInstance resources with filtering by group and phase
- `deckhouse_ListUnhealthyPods` — pods not in Running/Succeeded state

**Modules** (read-only)
- `deckhouse_ListModuleConfigs` — ModuleConfig resources with enabled/disabled filter

**Releases** (read-only)
- `deckhouse_ListDeckhouseReleases` — DeckhouseRelease resources with phase filter

**Nodes** (write)
- `deckhouse_CreateSSHCredentials` — create SSHCredentials resource
- `deckhouse_CreateStaticInstance` — create StaticInstance resource
- `deckhouse_AddWorkerNode` — composite: SSHCredentials → StaticInstance → wait for Running

## Tech Stack

- **Go 1.26** with [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- **Protobuf + [protoc-gen-mcp](https://github.com/easyp-tech/protoc-gen-mcp)** — proto-first tool generation
- **[EasyP](https://github.com/easyp-tech/easyp)** — proto linting, codegen, dependency management
- **client-go** — typed client for core resources, dynamic client for Deckhouse CRDs
- **Distroless** — minimal container image

## Prerequisites

- Go 1.26+
- [EasyP](https://github.com/easyp-tech/easyp) (`brew install easyp-tech/tap/easyp`)
- [go-task](https://taskfile.dev) (`brew install go-task`)
- Docker
- A Kubernetes cluster with Deckhouse CE installed

## Quick Start

```bash
# Generate protobuf code
task generate

# Build
task build

# Run tests
task test

# Build Docker image
task docker:build
```

## Development

### Available Tasks

```
task generate        # easyp mod download + easyp generate
task lint            # easyp lint
task build           # go build
task test            # go test ./...
task docker:build    # docker build
task docker:load     # kind load docker-image (for local Kind cluster)
task integration     # full integration test cycle
```

### Proto-First Workflow

All MCP tools are defined in `.proto` files under `proto/deckhouse/v1/`:

| File | Service | Purpose |
|------|---------|---------|
| `diagnostics.proto` | `DiagnosticsAPI` | Read-only cluster status, nodes, pods |
| `modules.proto` | `ModulesAPI` | ModuleConfig listing |
| `releases.proto` | `ReleasesAPI` | DeckhouseRelease listing |
| `nodes.proto` | `NodesAPI` | Node provisioning (SSHCredentials, StaticInstance) |

After editing `.proto` files, regenerate:

```bash
task generate
```

This produces `*.pb.go` (protobuf types) and `*.mcp.go` (MCP tool handler interfaces + registration).

### Implementing Handlers

Each handler implements a generated `*ToolHandler` interface:

```go
// Generated interface (example)
type DiagnosticsAPIToolHandler interface {
    GetClusterStatus(ctx context.Context, req *emptypb.Empty) (*GetClusterStatusResponse, error)
    ListNodes(ctx context.Context, req *ListNodesRequest) (*ListNodesResponse, error)
    // ...
}
```

Handlers live in `internal/handler/` and receive a `k8s.Client` interface for all Kubernetes operations.

### Testing

```bash
task test    # 38 unit tests, ~60s (includes polling tests)
```

Tests use a mock `k8s.Client` with function fields — no external mock libraries.

## Deployment

### RBAC

The server runs with least-privilege permissions:

- **Read**: `nodes`, `pods` (core); `nodegroups`, `staticinstances`, `moduleconfigs`, `deckhousereleases` (deckhouse.io CRDs)
- **Write**: `staticinstances`, `sshcredentials` (create only)

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LISTEN_ADDR` | `:8080` | HTTP listen address |

## Connecting an MCP Client

The server uses SSE transport. To connect:

1. **Open SSE stream**: `GET /sse` — receive `event: endpoint\ndata: /sse?sessionid=XXX`
2. **Initialize**: `POST /sse?sessionid=XXX` with JSON-RPC `initialize` request
3. **Confirm**: `POST` with `notifications/initialized`
4. **Call tools**: `POST` with `tools/call` — responses arrive via the SSE stream

Example with curl:

```bash
# Open SSE stream in background
curl -sN http://localhost:8080/sse > /tmp/sse.log &

# Get session endpoint (wait ~1s for first event)
ENDPOINT="http://localhost:8080$(grep '^data:' /tmp/sse.log | head -1 | sed 's/^data: *//')"

# Initialize
curl -s -X POST "$ENDPOINT" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":0,"method":"initialize","params":{
    "protocolVersion":"2024-11-05",
    "capabilities":{},
    "clientInfo":{"name":"demo","version":"1.0"}}}'

# Confirm initialized
curl -s -X POST "$ENDPOINT" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"notifications/initialized"}'

# List available tools
curl -s -X POST "$ENDPOINT" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'

# Call a tool
curl -s -X POST "$ENDPOINT" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{
    "name":"deckhouse_GetClusterStatus","arguments":{}}}'
```

Responses appear as `data:` lines in the SSE stream.

## Project Structure

```
proto/deckhouse/v1/          # .proto files — single source of truth
├── diagnostics.proto        # DiagnosticsAPI (5 RPCs)
├── modules.proto            # ModulesAPI (1 RPC)
├── releases.proto           # ReleasesAPI (1 RPC)
├── nodes.proto              # NodesAPI (3 RPCs)
├── config.proto             # ConfigAPI (stub, planned)
└── sources.proto            # SourcesAPI (stub, planned)
cmd/deckhouse-mcp/main.go   # SSE HTTP server entrypoint
internal/handler/            # Tool handler implementations
internal/k8s/client.go       # Kubernetes client interface
deploy/                      # Plain Kubernetes manifests
Taskfile.yml                 # Build tasks (go-task)
easyp.yaml                  # Proto config
Dockerfile                   # Multi-stage: golang → distroless
```

## License

[MIT](LICENSE)
