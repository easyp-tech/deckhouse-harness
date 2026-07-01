#!/usr/bin/env bash
# Set up the Kind cluster, apply CRD stubs + fixtures, and build the MCP binary.
# Wraps tests/integration/setup.sh — does NOT duplicate its logic.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

echo "==> Checking prerequisites..."
for cmd in kind kubectl go; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "ERROR: '$cmd' is required but not installed. Aborting." >&2
    exit 1
  fi
done

echo "==> Setting up Kind cluster and CRDs..."
cd "$PROJECT_ROOT"
bash tests/integration/setup.sh

echo "==> Applying fixtures..."
KUBECONFIG_PATH="$(cat tests/integration/.kube-context 2>/dev/null || echo "$HOME/.kube/config")"
KUBECONFIG="$KUBECONFIG_PATH" kubectl apply -f tests/integration/fixtures.yaml

echo "==> Building MCP binary..."
go build -o tests/integration/deckhouse-harness ./cmd/deckhouse-harness

echo ""
echo "==> Setup complete."
echo "    Binary:    $PROJECT_ROOT/tests/integration/deckhouse-harness"
echo "    KUBECONFIG: $KUBECONFIG_PATH"
echo ""
echo "Export before testing:"
echo "  export KUBECONFIG=$KUBECONFIG_PATH"
