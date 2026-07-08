# Deployment

`deckhouse-harness` is an **stdio** MCP server: a client launches the binary and
speaks newline-delimited JSON-RPC over stdin/stdout. There is no HTTP/SSE listener
and no long-running network service, so there is no Deployment or Service manifest.

The previous in-cluster SSE Deployment/Service were retired when the server moved to
the stdio-only `mcpruntime` runtime.

## Local use

Point your MCP client at the binary (built with `task build`):

```jsonc
{
  "mcpServers": {
    "deckhouse-harness": {
      "command": "/path/to/deckhouse-harness",
      "env": { "KUBECONFIG": "/path/to/kubeconfig" }
    }
  }
}
```

## In-cluster use

`rbac.yaml` provisions the `deckhouse-harness` ServiceAccount and the
`d8:deckhouse-harness` ClusterRole/Binding with the exact permissions the tools
need. Apply it and run the binary in a Pod that uses the ServiceAccount — the
server picks up in-cluster credentials automatically (`rest.InClusterConfig`):

```sh
kubectl apply -f deploy/rbac.yaml

# One-off interactive session over stdio (image built via `task docker:build`):
kubectl run deckhouse-harness -n d8-system --rm -i \
  --image=deckhouse-harness:local --image-pull-policy=Never \
  --overrides='{"spec":{"serviceAccountName":"deckhouse-harness"}}' \
  --restart=Never
```

Attach your MCP client to that process's stdio (or use a `kubectl exec -i`
transport into a long-lived Pod that runs the binary).
