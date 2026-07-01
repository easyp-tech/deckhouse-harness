#!/usr/bin/env bash
# Invoke a single deckhouse-harness tool over JSON-RPC stdio and print the response.
#
# Usage:
#   mcp-call.sh <tool_name> [args_json]
#
# Arguments:
#   tool_name   Full MCP tool name, e.g. deckhouse_GetClusterStatus
#   args_json   JSON object of tool arguments (default: {})
#
# Output:
#   Single JSON line: the JSON-RPC response object for id=1.
#   Exit 0 on success, non-zero if the binary fails to start or produces no output.
#
# Environment:
#   KUBECONFIG  Path to kubeconfig (defaults to tests/integration/.kube-context or ~/.kube/config)
#
# How it works:
#   Launches a fresh binary process, sends initialize + notifications/initialized + tools/call,
#   then closes stdin (EOF). The server exits on EOF. Logs are discarded (LOG_OUTPUT=discard).
set -euo pipefail

TOOL="${1:?Usage: mcp-call.sh <tool_name> [args_json]}"
ARGS="${2:-{}}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
BINARY="$PROJECT_ROOT/tests/integration/deckhouse-harness"
KUBE="${KUBECONFIG:-$(cat "$PROJECT_ROOT/tests/integration/.kube-context" 2>/dev/null || echo "$HOME/.kube/config")}"

if [[ ! -x "$BINARY" ]]; then
  echo "ERROR: binary not found at $BINARY — run setup.sh first." >&2
  exit 1
fi

# JSON-RPC messages
INIT='{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"e2e-agent","version":"1.0"}}}'
INITIALIZED='{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}'
CALL="{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"$TOOL\",\"arguments\":$ARGS}}"

# Send all three messages then close stdin so the server exits cleanly.
# Filter stdout to the line containing id=1 (the tool response).
RESPONSE="$(printf '%s\n%s\n%s\n' "$INIT" "$INITIALIZED" "$CALL" \
  | KUBECONFIG="$KUBE" LOG_OUTPUT=discard "$BINARY" 2>/dev/null \
  | grep '"id":1' || true)"

if [[ -z "$RESPONSE" ]]; then
  echo "ERROR: no response received for tool '$TOOL'" >&2
  exit 1
fi

printf '%s\n' "$RESPONSE"
