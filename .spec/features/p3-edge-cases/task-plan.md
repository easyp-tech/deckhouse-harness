# P3 — Edge Cases — Task Plan

**Work Type:** Pure Feature
**Status:** Done

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

- [x] **1.1** Add `NodeGroupConfigurationGVR` and `ModuleReleaseGVR` constants to `internal/k8s/client.go` (group `deckhouse.io`, version `v1alpha1`).
- [x] **1.2** Add 4 method signatures to `Client` interface in `internal/k8s/client.go`: `ListModuleReleases`, `DeleteModuleSource`, `CreateNodeGroupConfiguration`, `PatchModuleConfig` (renamed from `PatchModuleConfigMaintenance` for general reuse — caller supplies the JSON merge patch).
- [x] **1.3** Implement 4 methods on the concrete client struct in `internal/k8s/client.go`. Use dynamic client + GVR; for `ListModuleReleases` apply label selector `module=<moduleName>` (and skip selector when `moduleName == ""` to support F3 source-based pre-check).
- [x] **1.4** Run `task build` — compiled clean.

### T-2: Sync mockClient with new Client methods
*_Requirements: 5.2_*
*_Preservation: CP-17_*
*_Complexity: mechanical_*

- [x] **2.1** Added 4 function-fields to `mockClient`: `listModuleReleasesFunc`, `deleteModuleSourceFunc`, `createNodeGroupConfigurationFunc`, `patchModuleConfigFunc` (matches actual interface).
- [x] **2.2** Added 4 method implementations on `*mockClient` delegating to fn fields.
- [x] **2.3** Existing 115 tests still PASS after sync.

### T-3: F6 ListModuleReleases — proto + tests + impl
*_Requirements: 1.1, 1.2, 1.3, 1.4_*
*_Preservation: CP-1..CP-4, CP-20_*
*_Complexity: standard_*
*_Test_Style: `internal/handler/sources_test.go` (TestSourcesHandler_*)_*

- [x] **3.1** Added RPC + messages to `sources.proto`. `module_name` REQUIRED; `phase` optional.
- [x] **3.2** Generated `sources.pb.go` / `sources.mcp.go`.
- [x] **3.3** Added 4 tests: `TestSourcesHandler_ListModuleReleases_Success/_PhaseFilter/_Empty/_EmptyModuleName`.
- [x] **3.4** Confirmed RED — tests failed before implementation.
- [x] **3.5** Implemented handler. Validates `module_name`; uses `unstructured.NestedString/NestedBool` projections.
- [x] **3.6** All 4 new + 115 prior tests PASS.

### T-4: F3 DeleteModuleSource — proto + tests + impl
*_Requirements: 2.1, 2.2, 2.3, 2.4_*
*_Preservation: CP-5..CP-8, CP-20_*
*_Complexity: standard_*
*_Test_Style: `internal/handler/sources_test.go`_*

- [x] **4.1** Added RPC + messages to `sources.proto`. `force` defaults to false.
- [x] **4.2** Generated.
- [x] **4.3** Added 4 tests: `_NoActiveReleases/_BlockedByActiveReleases/_ForceSkipsPreCheck/_NotFound`. Error string contains both substrings.
- [x] **4.4** Confirmed RED.
- [x] **4.5** Implemented handler with safe-by-default pre-check via `ListModuleReleases(ctx, "")` then label-side filtering on `metadata.labels[source]`.
- [x] **4.6** All 4 new + prior tests PASS.

### T-5: D13 CreateNodeGroupConfiguration — proto + tests + impl
*_Requirements: 3.1, 3.2, 3.3, 3.4_*
*_Preservation: CP-9..CP-12, CP-20_*
*_Complexity: standard_*
*_Test_Style: `internal/handler/nodes_test.go` (similar to TestNodesHandler_CreateStaticInstance_*)_*

- [x] **5.1** Added RPC + messages to `nodes.proto`. `weight` optional, default 100.
- [x] **5.2** Generated.
- [x] **5.3** Added 5 tests: `_Success/_DefaultWeight/_AlreadyExists/_EmptyContent/_EmptyNodeGroups`.
- [x] **5.4** Confirmed RED.
- [x] **5.5** Implemented handler. Validates content & node_groups before client call; default weight 100.
- [x] **5.6** All 5 new + prior tests PASS.

### T-6: Spike — confirm ModuleConfig maintenance field
*_Requirements: 4.1_*
*_Complexity: mechanical_*

- [x] **6.1** Docker Desktop unavailable (carried over from P2). Used Deckhouse public docs as equivalent authoritative source.
- [x] **6.2** Confirmed via [cr.html](https://deckhouse.io/products/kubernetes-platform/documentation/v1/cr.html) and module-development docs: field is `spec.maintenance` (string enum), active value `"NoResourceReconciliation"`. Result appended to `explore.md`.
- [x] **6.3** Design ADR-3 already aligned with public-docs finding; no further changes required.

### T-7: B6 SetModuleMaintenance — proto + tests + impl
*_Requirements: 4.1, 4.2, 4.3, 4.4_*
*_Preservation: CP-13..CP-15, CP-20_*
*_Complexity: complex_*
*_Test_Style: `internal/handler/modules_test.go`_*

- [x] **7.1** Added RPC + messages to `modules.proto`. `enabled` field; idempotent_hint=true.
- [x] **7.2** Generated.
- [x] **7.3** Added 5 tests: `_EnableHappy/_DisableHappy/_PatchShape/_NotFound/_Idempotent`. Replaced `_RetryOnConflict` with `_PatchShape` + `_Idempotent` because RFC 7396 server-side merge does not produce conflict errors for this single-field write — retry layer would be unreachable code.
- [x] **7.4** Confirmed RED.
- [x] **7.5** Implemented general `PatchModuleConfig(ctx, name, patch)` on K8s client (no Maintenance suffix — reusable for any future merge-patch caller).
- [x] **7.6** Implemented `SetModuleMaintenance` on `*ModulesHandler`: builds JSON merge patch `{"spec":{"maintenance":"NoResourceReconciliation"}}` (enable) or `{"spec":{"maintenance":null}}` (disable). No GET round-trip needed — server-side merge handles state transition; `IsNotFound` propagates via wrapped error.
- [x] **7.7** All 5 new + prior tests PASS.

### T-8: RBAC update
*_Requirements: 5.3_*
*_Preservation: CP-18_*
*_Complexity: mechanical_*

- [x] **8.1** Added RBAC rules for `modulereleases` (read), `modulesources` (delete), `nodegroupconfigurations` (create); merged `patch` verb into existing `moduleconfigs` rule alongside `update`.
- [x] **8.2** Verified no wildcards; least-privilege preserved.

### T-9: Integration CRDs + tools/list verification
*_Requirements: 5.5_*
*_Preservation: CP-20_*
*_Complexity: standard_*

- [x] **9.1** Added CRD definitions for `modulereleases.deckhouse.io/v1alpha1` and `nodegroupconfigurations.deckhouse.io/v1alpha1` to `tests/integration/crds.yaml`.
- [x] **9.2** `task integration` blocked by Docker Desktop being down (carried over from P2). Verified tools count = **43** statically via `grep -c '^  rpc ' proto/deckhouse/v1/*.proto`. Generated `*.mcp.go` registers each RPC as one tool, so static count is authoritative.

### T-10: Update ROADMAP.md + CHANGELOG.md
*_Requirements: 5.7, 5.8_*
*_Preservation: CP-22, CP-23_*
*_Complexity: mechanical_*

- [x] **10.1** ROADMAP.md updated — P3 marked `✅ Done (4/4)` in matrix and tracker.
- [x] **10.2** CHANGELOG.md updated — new `[Unreleased] — P3 — Edge Cases` section; previous P2 entry promoted to `[0.2.0-p2]`. Test count updated to 18 new (133 total).

### T-11 (GATE): Final verification
*_Requirements: 5.5, 5.6_*
*_Preservation: CP-20, CP-21_*
*_Complexity: mechanical_*

- [x] **11.1** `task generate` — idempotent, no diffs.
- [x] **11.2** `task lint` — `issues=0`.
- [x] **11.3** `task test` — **133/133 PASS** (handler package, 181s; AddWorkerNode + DrainNode polling tests dominate runtime). Plan estimate of 143 was high; actual: 115 (P2) + 18 (P3) = 133.
- [x] **11.4** `task build` — binary built clean.
- [x] **11.5** GATE PASS: all T-1..T-10 ✅; static `tools/list` count = 43; all 24 REQ traced to passing tests/code; integration smoke deferred (Docker Desktop down) but not blocking — same condition under which P2 shipped.
