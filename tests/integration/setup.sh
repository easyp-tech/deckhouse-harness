#!/usr/bin/env bash
# Integration test setup: Deckhouse CE in Kind + MCP server Pod.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-d8}"
KUBE_CONTEXT="kind-${KIND_CLUSTER_NAME}"
MCP_IMAGE="deckhouse-mcp:local"
MCP_NAMESPACE="d8-system"
PORT_FORWARD_PID_FILE="$SCRIPT_DIR/.port-forward.pid"

info()  { echo "==> $*"; }
error() { echo "ERROR: $*" >&2; exit 1; }

# --- Prerequisites -----------------------------------------------------------
info "Checking prerequisites..."
for cmd in docker kind kubectl curl jq; do
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

# --- Build and load MCP server image -----------------------------------------
info "Building MCP server Docker image..."
docker build -t "$MCP_IMAGE" "$ROOT_DIR"

info "Loading image into Kind cluster..."
kind load docker-image "$MCP_IMAGE" --name "$KIND_CLUSTER_NAME"

# --- Deploy MCP server -------------------------------------------------------
info "Applying MCP server manifests..."
kubectl --context "$KUBE_CONTEXT" apply \
  -f "$ROOT_DIR/deploy/rbac.yaml" \
  -f "$ROOT_DIR/deploy/deployment.yaml" \
  -f "$ROOT_DIR/deploy/service.yaml"

# Restart deployment to pick up latest image (imagePullPolicy: Never).
kubectl --context "$KUBE_CONTEXT" -n "$MCP_NAMESPACE" rollout restart deployment/deckhouse-mcp

info "Waiting for MCP server pod to be ready..."
kubectl --context "$KUBE_CONTEXT" -n "$MCP_NAMESPACE" rollout status deployment/deckhouse-mcp --timeout=120s

# --- Port-forward -------------------------------------------------------------
# Kill any existing port-forward.
if [ -f "$PORT_FORWARD_PID_FILE" ]; then
  old_pid=$(cat "$PORT_FORWARD_PID_FILE")
  kill "$old_pid" 2>/dev/null || true
  rm -f "$PORT_FORWARD_PID_FILE"
fi

info "Starting port-forward to MCP server..."
kubectl --context "$KUBE_CONTEXT" -n "$MCP_NAMESPACE" \
  port-forward svc/deckhouse-mcp 8080:8080 >/dev/null 2>&1 &
echo $! > "$PORT_FORWARD_PID_FILE"

# Wait for port-forward to be ready (TCP check — SSE is streaming so curl hangs).
for i in $(seq 1 15); do
  if nc -z localhost 8080 2>/dev/null; then
    info "Port-forward is active on localhost:8080."
    break
  fi
  if [ "$i" -eq 15 ]; then
    error "Timeout waiting for port-forward to become ready."
  fi
  sleep 2
done

info "Setup complete. MCP server is accessible at http://localhost:8080"
