#!/usr/bin/env bash
# Integration test teardown: clean up test resources and port-forward.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
KUBE_CONTEXT="${KUBE_CONTEXT:-kind-d8}"
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-d8}"
PORT_FORWARD_PID_FILE="$SCRIPT_DIR/.port-forward.pid"

info()  { echo "==> $*"; }

# --- Kill port-forward --------------------------------------------------------
if [ -f "$PORT_FORWARD_PID_FILE" ]; then
  pid=$(cat "$PORT_FORWARD_PID_FILE")
  if kill "$pid" 2>/dev/null; then
    info "Killed port-forward (PID $pid)."
  fi
  rm -f "$PORT_FORWARD_PID_FILE"
else
  info "No port-forward PID file found."
fi

# --- Delete test resources ----------------------------------------------------
info "Cleaning up integration test resources..."
kubectl --context "$KUBE_CONTEXT" delete staticinstances \
  integration-test-si integration-test-worker integration-test-delete-si 2>/dev/null || true
kubectl --context "$KUBE_CONTEXT" delete sshcredentials \
  integration-test-creds integration-test-worker-creds 2>/dev/null || true
kubectl --context "$KUBE_CONTEXT" delete nodegroups \
  integration-test-ng 2>/dev/null || true
kubectl --context "$KUBE_CONTEXT" delete secret d8-cluster-configuration \
  -n kube-system 2>/dev/null || true

# --- Optionally delete MCP deployment ----------------------------------------
if [ "${DELETE_MCP:-}" = "true" ]; then
  info "Deleting MCP server deployment..."
  kubectl --context "$KUBE_CONTEXT" -n d8-system delete deployment/deckhouse-mcp svc/deckhouse-mcp 2>/dev/null || true
fi

# --- Optionally delete Kind cluster -------------------------------------------
if [ "${DELETE_CLUSTER:-}" = "true" ]; then
  info "Deleting Kind cluster '${KIND_CLUSTER_NAME}'..."
  kind delete cluster --name "$KIND_CLUSTER_NAME"
fi

info "Teardown complete."
