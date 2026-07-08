<!-- generated: 2026-07-07, template: bootstrap.md -->
# Deckhouse Harness — Documentation

This folder contains documentation to help LLMs and developers quickly understand the project context.

## Documentation Index

### Core
- [ARCHITECTURE.md](./ARCHITECTURE.md) — Layered architecture, data flow, key design decisions
- [PACKAGES.md](./PACKAGES.md) — Reference of all Go packages (tool/resource/prompt handlers, 36 k8s.Client methods)
- [DOMAIN.md](./DOMAIN.md) — Domain model: Deckhouse CRDs, MCP tools, K8s resources, lifecycle diagrams
- [CODE_STYLE.md](./CODE_STYLE.md) — Go and Proto conventions specific to this project

### Development
- [TOOLS.md](./TOOLS.md) — Task commands, code generation, CI/CD cheatsheet
- [TESTING.md](./TESTING.md) — Testing conventions, mock pattern, test structure

### API & Deployment
- [API.md](./API.md) — MCP API reference: 43 tools across 6 service blocks, plus Resources and Prompts (stdio transport, proto-first)
- [DEPLOYMENT.md](./DEPLOYMENT.md) — stdio run model, Docker image, ServiceAccount + RBAC

### Error Handling
- [ERRORS.md](./ERRORS.md) — Error wrapping conventions, propagation model

### Agent Rules
- [agent-rules.md](./agent-rules.md) — Mandatory rules for AI agents working on this project

## Quick Facts

| Aspect | Technology |
|--------|------------|
| **Language** | Go 1.26 |
| **Architecture** | Proto-first MCP server, handler pattern |
| **API** | MCP over stdio — protobuf-defined tools, resources, and prompts |
| **K8s client** | client-go v0.35.3 (typed + dynamic) |
| **Code generation** | protoc-gen-mcp v0.6.0 + easyp |
| **Runtime** | `protoc-gen-mcp/mcpruntime` (stdio; no `modelcontextprotocol/go-sdk`) |
| **Deployment** | stdio subprocess; in-cluster via ServiceAccount (no HTTP service) |
| **Auth** | ServiceAccount + ClusterRoleBinding (RBAC) |
| **Testing** | Standard `testing` package, function-field mocks |
| **MCP primitives** | 43 tools + 7 resources + 5 prompts — see [README.md](../README.md) |

## Project Structure

```
deckhouse-harness/
├── cmd/deckhouse-harness/    # Entry point (urfave/cli/v3) — stdio server, wires tools + resources + prompts
├── internal/
│   ├── handler/          # Tool (43) + resource + prompt handler implementations
│   └── k8s/              # Kubernetes client interface (36 methods) + implementation
├── proto/deckhouse/v1/   # .proto files (single source of truth) + generated *.pb.go, *.mcp.go
├── deploy/               # ServiceAccount + RBAC (no HTTP service) — see deploy/README.md
├── tests/integration/    # Integration test scripts (Kind cluster)
├── Dockerfile            # Multi-stage: golang:1.26 → distroless
├── Taskfile.yml          # go-task: generate, lint, build, test, docker
└── easyp.yaml            # Proto deps, lint rules, codegen plugins
```

## Running

```bash
# Code generation (proto → Go)
task generate   # easyp mod download && easyp generate

# Build
task build      # go build ./cmd/deckhouse-harness

# Tests (156 unit tests, ~3 min due to polling tests)
task test       # go test ./...

# Lint (proto)
task lint       # easyp lint

# Docker
task docker:build   # build image
task docker:load    # load into Kind cluster

# Integration tests (Kind)
task integration    # setup → test → teardown
```

## Ports

| Port | Component | Protocol |
|------|-----------|----------|
| —    | MCP stdio server | newline-delimited JSON-RPC on stdin/stdout |

Transport is stdio only — a client (Claude Desktop / Cursor / Claude Code) launches the binary as a subprocess. No network port.

## Key Interfaces

```go
// k8s.Client — all Kubernetes API operations (36 methods)
type Client interface { ... }  // internal/k8s/client.go

// Generated handler interfaces (one per proto service)
DiagnosticsAPIToolHandler   // proto/deckhouse/v1/diagnostics.mcp.go  (11 methods)
ModulesAPIToolHandler        // proto/deckhouse/v1/modules.mcp.go     (7 methods)
ReleasesAPIToolHandler       // proto/deckhouse/v1/releases.mcp.go    (3 methods)
NodesAPIToolHandler          // proto/deckhouse/v1/nodes.mcp.go       (13 methods)
ConfigAPIToolHandler         // proto/deckhouse/v1/config.mcp.go      (3 methods)
SourcesAPIToolHandler        // proto/deckhouse/v1/sources.mcp.go     (6 methods)
```

## Adding New Handlers

1. Add RPC to the appropriate `.proto` file in `proto/deckhouse/v1/`
2. Run `task generate` — updates `*.pb.go` and `*.mcp.go`
3. Implement the new method in `internal/handler/<service>.go`
4. Add new `k8s.Client` method(s) to `internal/k8s/client.go` if needed
5. Write unit tests in `internal/handler/<service>_test.go` using `mockClient`
6. Update RBAC in `deploy/rbac.yaml` if new K8s permissions are required
7. Handler is auto-registered via generated `pb.Register{Service}Tools()` — no changes needed in `main.go`
