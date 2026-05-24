# P3 — Edge Cases — Task Plan

**Work Type:** Pure Feature
**Status:** Draft

**Test Style Source:** Tier 2
- Evidence: `internal/handler/sources_test.go`, `nodes_test.go`, `modules_test.go`, `mock_client_test.go`
- Patterns: `Test{Handler}_{Method}_{Scenario}`, mock via function-fields, table-driven, no external libs

**Commands:**

| Action | Command | Source |
|--------|---------|--------|
| Test | `task test` | `Taskfile.yml` |
| Build | `task build` | `Taskfile.yml` |
| Lint | `task lint` | `Taskfile.yml` |
| Generate | `task generate` | `Taskfile.yml` |
| Integration | `task integration` | `Taskfile.yml` |

## Coverage Matrix

| Requirement | Task(s) | CP |
|-------------|---------|----|
| REQ-1.1 | T-3 | CP-1 |
| REQ-1.2 | T-3 | CP-2 |
| REQ-1.3 | T-3 | CP-3 |
| REQ-1.4 | T-3 | CP-4 |
| REQ-2.1 | T-4 | CP-5 |
| REQ-2.2 | T-4 | CP-6 |
| REQ-2.3 | T-4 | CP-7 |
| REQ-2.4 | T-4 | CP-8 |
| REQ-3.1 | T-5 | CP-9 |
| REQ-3.2 | T-5 | CP-10 |
| REQ-3.3 | T-5 | CP-11 |
| REQ-3.4 | T-5 | CP-12 |
| REQ-4.1 | T-6, T-7 | CP-13 |
| REQ-4.2 | T-7 | CP-13 |
| REQ-4.3 | T-7 | CP-14 |
| REQ-4.4 | T-7 | CP-15 |
| REQ-5.1 | T-1 | CP-16 |
| REQ-5.2 | T-2 | CP-17 |
| REQ-5.3 | T-8 | CP-18 |
| REQ-5.4 | T-1 | CP-19 |
| REQ-5.5 | T-11 | CP-20 |
| REQ-5.6 | T-11 | CP-21 |
| REQ-5.7 | T-10 | CP-22 |
| REQ-5.8 | T-10 | CP-23 |

## Tasks

### T-1: Extend k8s.Client interface (4 methods + 2 GVR)
*_Requirements: 5.1, 5.4_*
*_Complexity: mechanical_*

- [ ] **1.1** Add `NodeGroupConfigurationGVR` and `ModuleReleaseGVR` constants to `internal/k8s/client.go` (group `deckhouse.io`, version `v1alpha1`).
- [ ] **1.2** Add 4 method signatures to `Client` interface in `internal/k8s/client.go`: `ListModuleReleases`, `DeleteModuleSource`, `CreateNodeGroupConfiguration`, `PatchModuleConfigMaintenance`. Signatures — exactly as in design §2.3.
- [ ] **1.3** Implement 4 methods on the concrete client struct in `internal/k8s/client.go`. Use dynamic client + GVR; for `ListModuleReleases` apply label selector `module=<moduleName>`.
- [ ] **1.4** Run `task build` — CRITICAL: must compile (mockClient will fail until T-2; ignore that error).

### T-2: Sync mockClient with new Client methods
*_Requirements: 5.2_*
*_Preservation: CP-17_*
*_Complexity: mechanical_*

- [ ] **2.1** Add 4 function-fields to `mockClient` struct in `internal/handler/mock_client_test.go`: `listModuleReleasesFn`, `deleteModuleSourceFn`, `createNodeGroupConfigurationFn`, `patchModuleConfigMaintenanceFn`.
- [ ] **2.2** Add 4 method implementations on `*mockClient` that delegate to the fn fields (return helpful error if fn is nil).
- [ ] **2.3** Run `task test` — CRITICAL: must compile (existing 115 tests must still PASS).

### T-3: F6 ListModuleReleases — proto + tests + impl
*_Requirements: 1.1, 1.2, 1.3, 1.4_*
*_Preservation: CP-1..CP-4, CP-20_*
*_Complexity: standard_*
*_Test_Style: `internal/handler/sources_test.go` (TestSourcesHandler_*)_*

- [ ] **3.1** Edit `proto/deckhouse/v1/sources.proto`: add RPC `ListModuleReleases` and messages `ListModuleReleasesRequest` / `Response` / `ModuleReleaseInfo` per design §2.5. Annotate `read_only_hint: true`. Field `module_name` REQUIRED; `phase` optional.
- [ ] **3.2** Run `task generate` — verify `sources.pb.go` and `sources.mcp.go` updated; no errors.
- [ ] **3.3** Add 4 unit tests to `internal/handler/sources_test.go` (one per REQ-1.x). Use mock `listModuleReleasesFn`. Test names from design §2.8: `_Success`, `_PhaseFilter`, `_Empty`, `_EmptyModuleName`.
- [ ] **3.4** Run `task test` — these 4 tests MUST FAIL (handler not yet implemented). Confirm "no such method" or similar compile/runtime error.
- [ ] **3.5** Implement `ListModuleReleases` method on `*SourcesHandler` in `internal/handler/sources.go`: validate `module_name` non-empty (return error before client call); invoke `client.ListModuleReleases(ctx, req.ModuleName)`; apply optional phase filter via Go-side filtering; project unstructured → `ModuleReleaseInfo` (extract via `unstructured.NestedString` from metadata.name, labels[module], spec.version, labels[source], status.phase, spec.approved).
- [ ] **3.6** Run `task test` — all 4 new tests PASS; previous 115 tests still PASS.

### T-4: F3 DeleteModuleSource — proto + tests + impl
*_Requirements: 2.1, 2.2, 2.3, 2.4_*
*_Preservation: CP-5..CP-8, CP-20_*
*_Complexity: standard_*
*_Test_Style: `internal/handler/sources_test.go`_*

- [ ] **4.1** Edit `proto/deckhouse/v1/sources.proto`: add RPC `DeleteModuleSource` and messages per design §2.5. Annotate `destructive_hint: true`. Field `name` REQUIRED; `force` optional bool (default false).
- [ ] **4.2** Run `task generate`.
- [ ] **4.3** Add 4 unit tests to `sources_test.go`: `_NoForce_Blocked`, `_NoForce_OK`, `_Force_Bypass`, `_NotFound`. Use mocks `listModuleReleasesFn` (for blocked) and `deleteModuleSourceFn`. Error message for blocked must contain literal substring `"active releases"` and `"force=true"`.
- [ ] **4.4** Run `task test` — these 4 tests MUST FAIL initially.
- [ ] **4.5** Implement `DeleteModuleSource` on `*SourcesHandler`: IF `req.Force == nil || !*req.Force` → call `client.ListModuleReleases(ctx, "")` filtered by `source=<name>` label (NOTE: requires fetching all releases since labels iface returns all — implement Go-side filter on `labels["source"]`); if non-empty → return error `module source 'X' has N active releases (e.g., Y[, Z]); pass force=true to delete anyway`. Otherwise → `client.DeleteModuleSource(ctx, req.Name)`. Preserve `kerrors.IsNotFound` semantics.
- [ ] **4.6** Run `task test` — new 4 PASS; all prior PASS.

### T-5: D13 CreateNodeGroupConfiguration — proto + tests + impl
*_Requirements: 3.1, 3.2, 3.3, 3.4_*
*_Preservation: CP-9..CP-12, CP-20_*
*_Complexity: standard_*
*_Test_Style: `internal/handler/nodes_test.go` (similar to TestNodesHandler_CreateStaticInstance_*)_*

- [ ] **5.1** Edit `proto/deckhouse/v1/nodes.proto`: add RPC `CreateNodeGroupConfiguration` and messages. Annotate `destructive_hint: false` but write op (or `read_only_hint: false`). Fields: `name`, `content`, `node_groups` REQUIRED; `weight` optional int32.
- [ ] **5.2** Run `task generate`.
- [ ] **5.3** Add 5 unit tests to `nodes_test.go`: `_Success`, `_DefaultWeight`, `_AlreadyExists`, `_EmptyContent`, `_EmptyNodeGroups`.
- [ ] **5.4** Run `task test` — 5 new tests MUST FAIL initially.
- [ ] **5.5** Implement `CreateNodeGroupConfiguration` on `*NodesHandler` in `internal/handler/nodes.go`: validate `content != ""` and `len(node_groups) > 0` before client call; build `unstructured.Unstructured` with `apiVersion: deckhouse.io/v1alpha1`, `kind: NodeGroupConfiguration`, `metadata.name`, `spec.{content, nodeGroups, weight}`; if `weight` nil → default `int32(100)`; invoke `client.CreateNodeGroupConfiguration(ctx, obj)`; preserve `kerrors.IsAlreadyExists`.
- [ ] **5.6** Run `task test` — new 5 PASS; all prior PASS.

### T-6: Spike — confirm ModuleConfig maintenance field
*_Requirements: 4.1_*
*_Complexity: mechanical_*

- [ ] **6.1** Start Kind+Deckhouse CE if not running: `task integration:setup` (or skip if cluster already up).
- [ ] **6.2** Run `kubectl --context kind-d8 explain moduleconfig.spec | grep -iE 'maintenance|suspend|paus' > /tmp/p3-spike-maintenance.txt`. Also `kubectl --context kind-d8 get crd moduleconfigs.deckhouse.io -o yaml | grep -A2 -iE 'maintenance|suspend' >> /tmp/p3-spike-maintenance.txt`. Record exact field name and value type (bool/string/enum).
- [ ] **6.3** Update design assumption in `.spec/features/p3-edge-cases/design.md` ADR-3 with confirmed field name. If field doesn't exist (worst case) — document fallback: use annotation `deckhouse.io/maintenance: true` instead.

### T-7: B6 SetModuleMaintenance — proto + tests + impl
*_Requirements: 4.1, 4.2, 4.3, 4.4_*
*_Preservation: CP-13..CP-15, CP-20_*
*_Complexity: complex_*
*_Test_Style: `internal/handler/modules_test.go`_*

- [ ] **7.1** Edit `proto/deckhouse/v1/modules.proto`: add RPC `SetModuleMaintenance` and messages. Annotate `read_only_hint: false`. Fields: `module_name` REQUIRED; `enabled` REQUIRED bool.
- [ ] **7.2** Run `task generate`.
- [ ] **7.3** Add 5 unit tests to `modules_test.go`: `_Enter`, `_Exit`, `_Idempotent`, `_NotFound`, `_RetryOnConflict`.
- [ ] **7.4** Run `task test` — 5 new tests MUST FAIL initially.
- [ ] **7.5** Implement `PatchModuleConfigMaintenance` on concrete K8s client in `internal/k8s/client.go`: use field name from T-6 spike result; build JSON merge patch `{"spec":{"<field>": <value>}}`; on `kerrors.IsConflict` retry once with fresh GET.
- [ ] **7.6** Implement `SetModuleMaintenance` on `*ModulesHandler` in `internal/handler/modules.go`: GET ModuleConfig to determine previousState; if already in requested state → return success+previousState without patch (idempotency); else call `client.PatchModuleConfigMaintenance(ctx, name, enabled)`; preserve `kerrors.IsNotFound`.
- [ ] **7.7** Run `task test` — new 5 PASS; all prior PASS.

### T-8: RBAC update
*_Requirements: 5.3_*
*_Preservation: CP-18_*
*_Complexity: mechanical_*

- [ ] **8.1** Edit `deploy/rbac.yaml`. Add rules:
  - `apiGroups: ["deckhouse.io"], resources: ["modulereleases"], verbs: ["get", "list"]`
  - `apiGroups: ["deckhouse.io"], resources: ["modulesources"], verbs: ["delete"]` (extend existing modulesources rule if present)
  - `apiGroups: ["deckhouse.io"], resources: ["nodegroupconfigurations"], verbs: ["create"]`
  - `apiGroups: ["deckhouse.io"], resources: ["moduleconfigs"], verbs: ["patch"]` (extend if needed for B6).
- [ ] **8.2** Manually verify NO wildcards (`*`) introduced; verbs minimal.

### T-9: Integration CRDs + tools/list verification
*_Requirements: 5.5_*
*_Preservation: CP-20_*
*_Complexity: standard_*

- [ ] **9.1** Edit `tests/integration/crds.yaml`: add minimal CRD definitions for `modulereleases.deckhouse.io/v1alpha1` and `nodegroupconfigurations.deckhouse.io/v1alpha1` (cluster-scoped). Follow patterns of existing CRDs in the file.
- [ ] **9.2** Run `task integration` (full setup → test → teardown cycle). Verify `tools/list` returns 43 tools (count via `grep -c '"name"'` or equivalent in test output).

### T-10: Update ROADMAP.md + CHANGELOG.md
*_Requirements: 5.7, 5.8_*
*_Preservation: CP-22, CP-23_*
*_Complexity: mechanical_*

- [ ] **10.1** Edit `ROADMAP.md`: change Implementation Order row "P3 — Edge Cases | B6, D13, F3, F6 | 0/4" to "✅ Done (4/4)"; change Phase progress tracker line "- [ ] **P3 — Edge Cases** (0/4 handlers)" to "- [x] **P3 — Edge Cases** (4/4 handlers) — shipped".
- [ ] **10.2** Edit `CHANGELOG.md`: add new section at the top "## [Unreleased] — P3 — Edge Cases" listing 4 new tools (`deckhouse_ListModuleReleases`, `deckhouse_DeleteModuleSource`, `deckhouse_CreateNodeGroupConfiguration`, `deckhouse_SetModuleMaintenance`) and infrastructure changes (2 new GVRs, 4 Client methods, RBAC extensions).

### T-11 (GATE): Final verification
*_Requirements: 5.5, 5.6_*
*_Preservation: CP-20, CP-21_*
*_Complexity: mechanical_*

- [ ] **11.1** Run `task generate` — expect exit 0, no diffs after running (idempotent).
- [ ] **11.2** Run `task lint` — expect `issues=0`.
- [ ] **11.3** Run `task test` — expect ALL tests PASS (~143 tests including 18 new). Record exact count.
- [ ] **11.4** Run `task build` — expect binary built.
- [ ] **11.5** GATE: confirm all checkboxes in T-1..T-10 are checked; all 24 REQ traced to passing tests/code; if any failure — return to relevant T-x for fix.
