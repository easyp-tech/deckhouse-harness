<!-- generated: 2026-05-12, template: core.md -->
# Domain Model — Deckhouse MCP Server

## Core Concepts

The domain is Deckhouse Kubernetes Platform management. The server exposes K8s and Deckhouse CRD operations as MCP tools.

## Kubernetes Core Resources

| Resource | API | Operations | Description |
|----------|-----|------------|-------------|
| `Node` | `core/v1` | get, list, update (cordon) | Cluster node — status, conditions, labels, capacity |
| `Pod` | `core/v1` | list, delete (eviction) | Pod — phase, conditions, container statuses |
| `Event` | `core/v1` | list (filtered by node) | K8s events — for nodes, pods, etc. |
| `Secret` | `core/v1` | get (named: `d8-cluster-configuration`) | Used for `ClusterConfiguration` read |
| `Pod/log` | `core/v1` | get | Container logs (Deckhouse controller, any pod) |

## Deckhouse CRDs

| CRD | Group | Version | Resource | Operations |
|-----|-------|---------|----------|------------|
| `NodeGroup` | `deckhouse.io` | `v1` | `nodegroups` | get, list, create |
| `StaticInstance` | `deckhouse.io` | `v1alpha2` | `staticinstances` | get, list, create, delete |
| `SSHCredentials` | `deckhouse.io` | `v1alpha2` | `sshcredentials` | create |
| `ModuleConfig` | `deckhouse.io` | `v1alpha1` | `moduleconfigs` | get, list, update |
| `DeckhouseRelease` | `deckhouse.io` | `v1alpha1` | `deckhouserelease` | get, list, patch |

All CRDs are handled as `unstructured.Unstructured` — no generated Go types.

### NodeGroup
Controls a group of homogeneous nodes. Key status fields:
- `.status.ready` — count of ready nodes
- `.status.nodes` — total node count
- `.spec.nodeType` — `Static`, `CloudEphemeral`, `CloudPermanent`, `CloudStatic`
- `.spec.disruptions.approvalMode` — `Automatic` or `Manual`

### StaticInstance
Represents a static (bare-metal / VM) machine to be registered as a K8s node. Key fields:
- `.spec.address` — SSH address
- `.spec.credentialsRef.name` — reference to `SSHCredentials`
- `.spec.labels` — labels for NodeGroup binding via `labelSelector`
- `.status.currentStatus.phase` — lifecycle phase (`Pending`, `Bootstrapping`, `Running`, `Error`)

### SSHCredentials
Stores SSH credentials for accessing static instances. Key fields:
- `.spec.user` — SSH user
- `.spec.privateSSHKey` — base64-encoded private key (encoded by handler, never by client)
- `.spec.sshExtraArgs` — extra SSH options
- `.spec.sshPort` — SSH port (default: 22)
- `.spec.sudoPassword` — base64-encoded sudo password (optional)

### ModuleConfig
Controls Deckhouse module configuration. Key fields:
- `.spec.enabled` — whether the module is enabled (writable via `EnableModule`/`DisableModule`)
- `.spec.version` — config schema version
- `.spec.settings` — module-specific settings (free-form object)
- `.status.status` — current module status (`Ready`, `Error`, etc.)
- `.status.source` — ModuleSource providing the module
- `.status.updatePolicy` — applied ModuleUpdatePolicy name

### DeckhouseRelease
Represents a Deckhouse release available for update. Key fields:
- `.spec.version` — release version string
- `.spec.requirements` — map of requirements (e.g., `k8s: ">=1.28"`)
- `.spec.changelogLink` — URL to release changelog
- `.metadata.annotations["release.deckhouse.io/approved"]` — approval annotation (patchable via `ApproveRelease`)
- `.status.phase` — `Pending`, `Deployed`, `Superseded`, `Skipped`
- `.status.transitionTime` — timestamp of last phase transition

## MCP Tools Domain

MCP tools are organized in 6 blocks (A–F), corresponding to proto service files:

| Block | Domain | Tools (implemented) |
|-------|--------|---------------------|
| A: Diagnostics | Read-only cluster observability | `GetClusterStatus`, `ListNodes`, `ListNodeGroups`, `ListStaticInstances`, `ListUnhealthyPods`, `GetNode`, `GetNodeGroup`, `GetDeckhouseLogs` |
| B: Modules | ModuleConfig management | `ListModuleConfigs`, `GetModuleConfig`, `EnableModule`, `DisableModule` |
| C: Releases | Deckhouse release management | `ListDeckhouseReleases`, `GetDeckhouseRelease`, `ApproveRelease` |
| D: Nodes | Node lifecycle (static nodes) | `CreateSSHCredentials`, `CreateStaticInstance`, `AddWorkerNode`, `DeleteStaticInstance`, `RemoveNode`, `CreateNodeGroup`, `WaitNodeReady` |
| E: Config | Cluster configuration | `GetClusterConfiguration` |
| F: Sources | ModuleSource and update policies | (not yet implemented) |

**Total: 23 implemented tools** (P0 + P1 complete).

## Tool Naming

Tools follow the pattern: `deckhouse_{MethodName}` (MCP namespace = `deckhouse`).  
Example: `DiagnosticsAPI.GetClusterStatus` → MCP tool `deckhouse_GetClusterStatus`.

## Node Lifecycle (Static Nodes)

### Adding a node (`AddWorkerNode` composite)

```
SSHCredentials (created)
       ↓
StaticInstance (created, refs SSHCredentials + NodeGroup labels)
       ↓
status.currentStatus.phase = "Pending"
       ↓  (Deckhouse bootstraps node via SSH)
status.currentStatus.phase = "Bootstrapping"
       ↓
status.currentStatus.phase = "Running"
       ↓
K8s Node object appears in cluster
```

`AddWorkerNode` automates this entire flow with polling (30s interval, 15min timeout).

### Removing a node (`RemoveNode` composite)

```
Node cordoned (unschedulable=true)
       ↓
Non-DaemonSet pods evicted
       ↓
StaticInstance deleted
       ↓
Deckhouse cleans up the node
```

`RemoveNode` performs cordon → drain → delete as a single operation. Drain is optional (`drain` field, default `true`).

## Release Lifecycle

```
DeckhouseRelease created (by Deckhouse operator)
       ↓
status.phase = "Pending"
       ↓  (if manual approval mode)
annotation "release.deckhouse.io/approved" = "true"  ← ApproveRelease tool
       ↓
status.phase = "Deployed"
       ↓  (newer release deployed)
status.phase = "Superseded"
```

## Module Lifecycle

```
ModuleConfig exists (spec.enabled = true/false)
       ↓
EnableModule  → sets spec.enabled = true
DisableModule → sets spec.enabled = false
       ↓
Deckhouse reconciles, activates/deactivates module
       ↓
status.status = "Ready" or "Error"
```
