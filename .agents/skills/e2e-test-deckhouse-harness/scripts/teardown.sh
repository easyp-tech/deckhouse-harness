#!/usr/bin/env bash
# Remove all e2e test resources created during the test run.
# Identifies resources by the label: e2e-test=true
# The Kind cluster itself is NOT deleted.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
KUBE="${KUBECONFIG:-$(cat "$PROJECT_ROOT/tests/integration/.kube-context" 2>/dev/null || echo "$HOME/.kube/config")}"

export KUBECONFIG="$KUBE"

echo "==> Removing e2e test resources (label: e2e-test=true)..."

# Deckhouse CRDs — cluster-scoped
for resource in \
  staticinstances.deckhouse.io \
  sshcredentials.deckhouse.io \
  nodegroups.deckhouse.io \
  modulesources.deckhouse.io \
  moduleupdatepolicies.deckhouse.io \
  nodegroupconfigurations.deckhouse.io; do
  kubectl delete "$resource" -l e2e-test=true --ignore-not-found
done

echo "==> Teardown complete. Kind cluster is still running."
echo "    To delete the cluster: kind delete cluster --name d8"
