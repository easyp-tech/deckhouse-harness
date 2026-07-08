<!-- generated: 2026-07-07, template: deployment.md -->
# Deployment — Deckhouse Harness

## 1. Overview

`deckhouse-harness` is an **stdio** MCP server: a client launches the binary and
speaks newline-delimited JSON-RPC over stdin/stdout. There is no HTTP/SSE listener
and no long-running network service, so there are no Deployment or Service
manifests — only a ServiceAccount + RBAC for in-cluster runs.

- **Local**: an MCP client (Claude Desktop, Cursor, Claude Code) launches the
  binary with a `KUBECONFIG` pointing at the cluster.
- **In-cluster**: run the binary in a Pod using the `deckhouse-harness`
  ServiceAccount; it picks up in-cluster credentials via `rest.InClusterConfig()`.
  A client attaches to that process's stdio (e.g. `kubectl exec -i`).

## 2. Environments

| Environment | Host | Purpose |
|-------------|------|---------|
| Local (Kind) | `localhost` | Development + integration tests |
| In-cluster | `d8-system` NS | Live, managing the Deckhouse cluster |

No staging environment. Local testing uses Kind (`task integration`).

## 3. Docker

### Dockerfile (multi-stage)

```dockerfile
FROM golang:1.26 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /deckhouse-mcp ./cmd/deckhouse-harness

FROM gcr.io/distroless/static-debian12
COPY --from=builder /deckhouse-mcp /deckhouse-mcp
USER nonroot:nonroot
ENTRYPOINT ["/deckhouse-mcp"]
```

- **Builder**: `golang:1.26` — dependencies cached in a separate layer
- **Runtime**: `distroless/static-debian12` — minimal, no shell
- **CGO disabled** — fully static binary
- **Runs as nonroot** — security best practice
- **No exposed ports** — the server communicates over stdio, not the network

### Useful commands

```bash
task docker:build           # docker build -t deckhouse-harness:local .
task docker:load            # kind load docker-image deckhouse-harness:local --name d8
```

## 4. Kubernetes Manifests (`deploy/`)

Only RBAC is shipped; see `deploy/README.md` for the stdio run model.

### RBAC (`rbac.yaml`)

ServiceAccount `deckhouse-harness` in `d8-system` with a `ClusterRole` +
`ClusterRoleBinding`.

**Current permissions (all 43 tools, P0–P3):**

| Resource | APIGroup | Verbs |
|----------|----------|-------|
| `nodes`, `pods`, `events` | `""` (core) | `get`, `list` |
| `pods/log` | `""` (core) | `get`, `list` |
| `nodes` | `""` (core) | `update`, `patch` (cordon/uncordon/drain) |
| `pods/eviction` | `""` (core) | `create` (drain) |
| `secrets` (named: `d8-cluster-configuration`) | `""` (core) | `get`, `update` |
| `nodegroups`, `staticinstances`, `moduleconfigs`, `deckhousereleases`, `modules`, `modulesources`, `moduleupdatepolicies`, `modulereleases` | `deckhouse.io` | `get`, `list` |
| `staticinstances`, `sshcredentials`, `nodegroups`, `modulesources`, `moduleupdatepolicies`, `nodegroupconfigurations` | `deckhouse.io` | `create` |
| `staticinstances`, `sshcredentials`, `nodegroups`, `modulesources` | `deckhouse.io` | `delete` |
| `moduleconfigs` | `deckhouse.io` | `update`, `patch` |
| `deckhousereleases` | `deckhouse.io` | `patch` |

**Expanding RBAC for new handlers:**

When adding new handlers, expand `deploy/rbac.yaml` with least-privilege permissions for the resources each new tool touches. Always add only the specific verbs needed (no wildcards).

## 5. Lifecycle

The server is a per-session subprocess, not a long-running service: it starts when
a client connects its stdio and exits when stdin closes or on `SIGINT`/`SIGTERM`
(graceful shutdown via context cancellation). It is stateless (no persistent
state, no DB), so restarts are safe at any time.

## 6. Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KUBECONFIG` | `~/.kube/config` | Path to kubeconfig (or in-cluster config inside a Pod) |
| `LOG_LEVEL` | `INFO` | Log verbosity: `DEBUG`, `INFO`, `WARN`, `ERROR` |
| `LOG_OUTPUT` | `stderr` | Log destination: `stderr`, `file`, `discard` |
| `LOG_FILE` | — | Log file path (required when `LOG_OUTPUT=file`) |
