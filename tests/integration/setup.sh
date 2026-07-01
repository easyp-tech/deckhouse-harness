#!/usr/bin/env bash
# Integration test setup: Kind + Deckhouse CE + build local MCP binary.
#
# The MCP server runs as a local stdio process during tests — no Docker image,
# no Kubernetes deployment, no port-forward.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-d8}"
KUBE_CONTEXT="kind-${KIND_CLUSTER_NAME}"
BINARY_PATH="$SCRIPT_DIR/deckhouse-mcp"

info()  { echo "==> $*"; }
error() { echo "ERROR: $*" >&2; exit 1; }

# --- Prerequisites -----------------------------------------------------------
info "Checking prerequisites..."
for cmd in kind kubectl jq go; do
  command -v "$cmd" >/dev/null 2>&1 || error "$cmd is not installed"
done
info "All prerequisites satisfied."

# --- Kind cluster with Deckhouse CE ------------------------------------------
if kind get clusters 2>/dev/null | grep -q "^${KIND_CLUSTER_NAME}$"; then
  info "Kind cluster '${KIND_CLUSTER_NAME}' already exists, reusing."
else
  info "Creating Kind cluster with Deckhouse CE (this takes ~15 minutes)..."
  bash -c "$(curl -Ls https://raw.githubusercontent.com/deckhouse/deckhouse/main/tools/kind-d8.sh)"
fi

# Wait for Deckhouse to be ready (moduleconfig 'deckhouse' must exist).
info "Waiting for Deckhouse to be ready..."
for i in $(seq 1 60); do
  if kubectl --context "$KUBE_CONTEXT" get moduleconfigs deckhouse >/dev/null 2>&1; then
    info "Deckhouse is ready."
    break
  fi
  if [ "$i" -eq 60 ]; then
    error "Timeout waiting for Deckhouse to become ready."
  fi
  sleep 10
done

# --- Build MCP server binary -------------------------------------------------
info "Building MCP server binary..."
go build -o "$BINARY_PATH" "$ROOT_DIR/cmd/deckhouse-mcp"
info "Binary built at $BINARY_PATH"

# Export variables for test.sh (sourced via Taskfile env).
echo "$KUBE_CONTEXT" > "$SCRIPT_DIR/.kube-context"
echo "$BINARY_PATH" > "$SCRIPT_DIR/.binary-path"

info "Setup complete. Run 'task integration:test' to start tests."
