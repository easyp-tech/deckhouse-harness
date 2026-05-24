<!-- generated: 2026-05-12, template: api.md -->
# API Reference — Deckhouse MCP Server

## 1. Overview

**Protocol:** Model Context Protocol (MCP) over Server-Sent Events (HTTP/SSE).  
**Tool definitions:** Generated from protobuf via `protoc-gen-mcp`. All tools use ProtoJSON encoding.  
**Transport:** `POST /sse` — SSE connection; MCP messages multiplexed over stream.  
**Auth:** None at the MCP level — Kubernetes RBAC enforces access at the Pod/SA level.  
**Base URL:** `http://<service>:8080` (in-cluster: `deckhouse-mcp.d8-system.svc.cluster.local:8080`)

## 2. Tool Naming

All tools use the namespace prefix `deckhouse_`:

```
{service_rpc_name}  →  deckhouse_{MethodName}
DiagnosticsAPI.GetClusterStatus  →  deckhouse_GetClusterStatus
```

## 3. Tools Summary

| # | Tool | Block | Type | Description |
|---|------|-------|------|-------------|
| 1 | `deckhouse_GetClusterStatus` | Diagnostics | read | High-level cluster health summary |
| 2 | `deckhouse_ListNodes` | Diagnostics | read | All cluster nodes with status |
| 3 | `deckhouse_ListNodeGroups` | Diagnostics | read | All Deckhouse NodeGroups |
| 4 | `deckhouse_ListStaticInstances` | Diagnostics | read | All StaticInstance resources |
| 5 | `deckhouse_ListUnhealthyPods` | Diagnostics | read | Pods not Running/Succeeded |
| 6 | `deckhouse_GetNode` | Diagnostics | read | Detailed node info + events |
| 7 | `deckhouse_GetNodeGroup` | Diagnostics | read | Single NodeGroup detail |
| 8 | `deckhouse_GetDeckhouseLogs` | Diagnostics | read | Deckhouse controller pod logs |
| 9 | `deckhouse_ListModuleConfigs` | Modules | read | All ModuleConfig objects |
| 10 | `deckhouse_GetModuleConfig` | Modules | read | Single ModuleConfig detail |
| 11 | `deckhouse_EnableModule` | Modules | write | Enable a module (idempotent) |
| 12 | `deckhouse_DisableModule` | Modules | write | Disable a module (idempotent) |
| 13 | `deckhouse_ListDeckhouseReleases` | Releases | read | All DeckhouseRelease objects |
| 14 | `deckhouse_GetDeckhouseRelease` | Releases | read | Single release detail |
| 15 | `deckhouse_ApproveRelease` | Releases | write | Approve pending release |
| 16 | `deckhouse_CreateSSHCredentials` | Nodes | write | Create SSHCredentials resource |
| 17 | `deckhouse_CreateStaticInstance` | Nodes | write | Create StaticInstance resource |
| 18 | `deckhouse_AddWorkerNode` | Nodes | write (composite) | SSH + StaticInstance + wait |
| 19 | `deckhouse_DeleteStaticInstance` | Nodes | write | Delete a StaticInstance |
| 20 | `deckhouse_RemoveNode` | Nodes | write (composite) | Cordon + drain + delete SI |
| 21 | `deckhouse_CreateNodeGroup` | Nodes | write | Create a NodeGroup |
| 22 | `deckhouse_WaitNodeReady` | Nodes | read (polling) | Poll until SI reaches Running |
| 23 | `deckhouse_GetClusterConfiguration` | Config | read | Read ClusterConfiguration YAML |

## 4. Tools Reference

### Block A — Diagnostics (8 tools, read-only)

All diagnostics tools have `read_only_hint: true`.

---

#### `deckhouse_GetClusterStatus`

Returns a high-level cluster health summary.

**Input:** `{}` (empty)

**Output:**
```json
{
  "nodes": { "ready": 3, "total": 4 },
  "nodeGroups": [
    { "name": "workers", "ready": 2, "total": 3 }
  ],
  "erroredModules": ["my-module"],
  "pendingReleases": ["v1.65.0"]
}
```

---

#### `deckhouse_ListNodes`

Lists all cluster nodes with status and details.

**Input:** `{}`

**Output:**
```json
{
  "nodes": [
    {
      "name": "node-1",
      "ready": true,
      "roles": ["worker"],
      "internalIp": "192.168.1.10",
      "kubeletVersion": "v1.29.0"
    }
  ]
}
```

---

#### `deckhouse_ListNodeGroups`

Lists all Deckhouse NodeGroups.

**Input:** `{}`

**Output:**
```json
{
  "nodeGroups": [
    { "name": "workers", "ready": 2, "nodes": 3, "nodeType": "Static" }
  ]
}
```

---

#### `deckhouse_ListStaticInstances`

Lists all StaticInstance resources.

**Input:** `{}`

**Output:**
```json
{
  "staticInstances": [
    {
      "name": "worker-01",
      "address": "10.0.0.5",
      "phase": "Running",
      "nodeName": "worker-01"
    }
  ]
}
```

---

#### `deckhouse_ListUnhealthyPods`

Lists pods that are not Running/Succeeded.

**Input:**
```json
{ "namespace": "my-namespace" }
```
`namespace` is optional — omit to search all namespaces.

**Output:**
```json
{
  "pods": [
    {
      "name": "my-pod-abc",
      "namespace": "default",
      "phase": "CrashLoopBackOff",
      "reason": "OOMKilled"
    }
  ]
}
```

---

#### `deckhouse_GetNode`

Returns detailed information about a single node including conditions, capacity, allocatable resources, and recent events.

**Input:**
```json
{ "name": "worker-01" }
```

**Output:**
```json
{
  "name": "worker-01",
  "ready": true,
  "roles": ["worker"],
  "internalIp": "192.168.1.10",
  "kubeletVersion": "v1.29.0",
  "conditions": [...],
  "capacity": { "cpu": "4", "memory": "8Gi" },
  "events": [...]
}
```

---

#### `deckhouse_GetNodeGroup`

Returns full spec and status of a single NodeGroup.

**Input:**
```json
{ "name": "workers" }
```

**Output:**
```json
{
  "name": "workers",
  "ready": 2,
  "nodes": 3,
  "nodeType": "Static",
  "conditions": [...]
}
```

---

#### `deckhouse_GetDeckhouseLogs`

Returns logs from the Deckhouse controller pod (`d8-system/deckhouse-*`).

**Input:**
```json
{
  "tail": 100,
  "container": "deckhouse"
}
```
Both fields are optional. `tail` defaults to 100.

**Output:**
```json
{
  "logs": "...",
  "podName": "deckhouse-7b4d5c8f-abc12"
}
```

---

### Block B — Modules (4 tools)

---

#### `deckhouse_ListModuleConfigs`

Lists all ModuleConfig objects with optional filter.

**Input:**
```json
{ "enabled": true }
```
`enabled` is optional — omit to list all.

**Output:**
```json
{
  "modules": [
    {
      "name": "cert-manager",
      "enabled": true,
      "version": "1",
      "source": "",
      "updatePolicy": "",
      "statusMessage": "Ready"
    }
  ]
}
```

---

#### `deckhouse_GetModuleConfig`

Returns full spec and status of a single ModuleConfig.

**Input:**
```json
{ "name": "cert-manager" }
```

**Output:**
```json
{
  "name": "cert-manager",
  "enabled": true,
  "version": 1,
  "settings": { "key": "value" },
  "statusMessage": "Ready"
}
```

---

#### `deckhouse_EnableModule`

Enables a Deckhouse module by setting `spec.enabled=true`. Idempotent — safe to call when already enabled.

**Input:**
```json
{ "name": "cert-manager" }
```

**Output:**
```json
{
  "success": true,
  "previousState": false
}
```

---

#### `deckhouse_DisableModule`

Disables a Deckhouse module by setting `spec.enabled=false`. Idempotent — safe to call when already disabled.

**Input:**
```json
{ "name": "cert-manager" }
```

**Output:**
```json
{
  "success": true,
  "previousState": true
}
```

---

### Block C — Releases (3 tools)

---

#### `deckhouse_ListDeckhouseReleases`

Lists all DeckhouseRelease objects with optional phase filter.

**Input:**
```json
{ "phase": "DECKHOUSE_RELEASE_PHASE_PENDING" }
```
`phase` is optional — omit to list all. Enum values: `PENDING`, `DEPLOYED`, `SUPERSEDED`, `SKIPPED`.

**Output:**
```json
{
  "releases": [
    {
      "name": "v1.74.0",
      "version": "v1.74.0",
      "phase": "Pending",
      "approved": false,
      "transitionTime": "2026-05-01T12:00:00Z",
      "changelogLink": "https://deckhouse.ru/changelog/v1.74.0"
    }
  ]
}
```

---

#### `deckhouse_GetDeckhouseRelease`

Returns full spec and status of a single DeckhouseRelease.

**Input:**
```json
{ "version": "v1.74.0" }
```

**Output:**
```json
{
  "name": "v1.74.0",
  "version": "v1.74.0",
  "phase": "Pending",
  "approved": false,
  "transitionTime": "2026-05-01T12:00:00Z",
  "changelogLink": "https://deckhouse.ru/changelog/v1.74.0",
  "requirements": { "k8s": ">=1.28" }
}
```

---

#### `deckhouse_ApproveRelease`

Approves a pending Deckhouse release by patching the `release.deckhouse.io/approved` annotation to `"true"`. Idempotent.

**Input:**
```json
{ "version": "v1.74.0" }
```

**Output:**
```json
{
  "success": true,
  "previousApproved": false
}
```

---

### Block D — Nodes (7 tools, write)

All node write tools have `destructive_hint: true` or `read_only_hint: false`.

---

#### `deckhouse_CreateSSHCredentials`

Creates an SSHCredentials resource. Private key and sudo password are accepted as plain text — base64 encoding happens inside the handler.

**Input:**
```json
{
  "name": "worker-1-ssh",
  "user": "ubuntu",
  "privateKey": "-----BEGIN RSA PRIVATE KEY-----\n...",
  "port": 22,
  "sshExtraArgs": "-o StrictHostKeyChecking=no",
  "sudoPassword": "secret"
}
```
`port`, `sshExtraArgs`, and `sudoPassword` are optional.

**Output:**
```json
{ "name": "worker-1-ssh" }
```

---

#### `deckhouse_CreateStaticInstance`

Creates a StaticInstance resource linking a machine to an SSHCredentials object.

**Input:**
```json
{
  "name": "worker-01",
  "address": "192.168.1.100",
  "credentialsRef": "worker-1-ssh",
  "labels": { "role": "worker" }
}
```

**Output:**
```json
{
  "name": "worker-01",
  "address": "192.168.1.100",
  "credentialsRef": "worker-1-ssh",
  "labels": { "role": "worker" }
}
```

---

#### `deckhouse_AddWorkerNode`

Composite operation: creates SSHCredentials + StaticInstance, then polls until node is bootstrapped (default 15 min timeout).

**Input:**
```json
{
  "address": "192.168.1.100",
  "sshUser": "ubuntu",
  "privateKey": "-----BEGIN OPENSSH PRIVATE KEY-----\n...",
  "nodeGroup": "workers",
  "nodeName": "worker-1",
  "sshPort": 22,
  "waitReady": true,
  "timeoutSeconds": 900
}
```
`nodeName`, `sshPort`, `waitReady`, `timeoutSeconds` are optional.

**Output:**
```json
{
  "nodeName": "worker-1",
  "sshCredentialsName": "worker-1-ssh",
  "staticInstanceName": "worker-1",
  "phase": "Running",
  "elapsed": "2m30s",
  "timedOut": false
}
```
If timed out: `"timedOut": true`, `phase` reflects last observed status.

---

#### `deckhouse_DeleteStaticInstance`

Deletes a StaticInstance resource by name. Deckhouse will gracefully clean up the node.

**Input:**
```json
{ "name": "worker-01" }
```

**Output:**
```json
{ "success": true }
```

---

#### `deckhouse_RemoveNode`

Composite: cordon the node, evict all non-DaemonSet pods, then delete the associated StaticInstance. Static nodes only.

**Input:**
```json
{
  "name": "worker-01",
  "drain": true
}
```
`drain` is optional (default: `true`).

**Output:**
```json
{
  "drained": true,
  "deleted": true
}
```

---

#### `deckhouse_CreateNodeGroup`

Creates a new NodeGroup resource. In Deckhouse CE only `Static` nodeType is supported.

**Input:**
```json
{
  "name": "workers",
  "nodeType": "Static",
  "count": 3,
  "labels": { "node-role.kubernetes.io/worker": "" },
  "disruptions": "Automatic",
  "maxPodsPerNode": 110
}
```
`count`, `labels`, `disruptions`, `maxPodsPerNode` are optional.

**Output:**
```json
{
  "name": "workers",
  "nodeType": "Static",
  "count": 3
}
```

---

#### `deckhouse_WaitNodeReady`

Polls StaticInstance status until it reaches `Running` phase or timeout.

**Input:**
```json
{
  "name": "worker-01",
  "timeoutSeconds": 900,
  "intervalSeconds": 30
}
```
`timeoutSeconds` and `intervalSeconds` are optional (defaults: 900, 30).

**Output:**
```json
{
  "phase": "Running",
  "elapsed": "45s",
  "timedOut": false
}
```

---

### Block E — Configuration (1 tool)

---

#### `deckhouse_GetClusterConfiguration`

Reads the ClusterConfiguration from the `d8-cluster-configuration` Secret in `kube-system`. Returns raw YAML.

**Input:** `{}`

**Output:**
```json
{
  "configuration": "apiVersion: deckhouse.io/v1\nkind: ClusterConfiguration\n..."
}
```

---

## 5. Error Format

MCP tool errors are returned as standard MCP `CallToolResult` with `isError: true`. The `content` text field contains a human-readable message:

```
"listing nodes: connection refused"
"privateKey is required"
"not found: StaticInstance worker-01"
```

No structured error codes — errors are descriptive strings from `fmt.Errorf`.

## 6. Transport Details

**Connection:** The MCP client opens an SSE connection via HTTP GET to `/sse`. The server sends MCP messages as SSE events.

**Each request:** MCP client sends a tool call message. Server processes it synchronously and sends the result as an SSE event.

**Polling tools** (`AddWorkerNode`, `WaitNodeReady`): block the SSE connection for up to 15 minutes while polling K8s. The client must not timeout the connection.

**Server-side timeout:** HTTP server uses `ReadHeaderTimeout: 10s`. No per-request timeout — callers use context cancellation.

## 7. Proto / Schema

**`.proto` file locations:** `proto/deckhouse/v1/`

| File | Service | RPCs |
|------|---------|------|
| `diagnostics.proto` | `DiagnosticsAPI` | 8 RPCs |
| `modules.proto` | `ModulesAPI` | 4 RPCs |
| `releases.proto` | `ReleasesAPI` | 3 RPCs |
| `nodes.proto` | `NodesAPI` | 7 RPCs |
| `config.proto` | `ConfigAPI` | 1 RPC |
| `sources.proto` | `SourcesAPI` | 0 RPCs (stub) |

**Code generation command:**
```bash
task generate  # easyp mod download && easyp generate
```

**Tool registration (auto-generated):**
```go
pb.RegisterDiagnosticsAPITools(server, diagnosticsHandler)
pb.RegisterModulesAPITools(server, modulesHandler)
pb.RegisterReleasesAPITools(server, releasesHandler)
pb.RegisterNodesAPITools(server, nodesHandler)
pb.RegisterConfigAPITools(server, configHandler)
```
