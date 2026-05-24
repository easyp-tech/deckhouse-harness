# P2 — Advanced Management: Task Plan

**Статус:** Draft
**Дата:** 2026-05-13
**Входные артефакты:** `explore.md`, `requirements.md`, `design.md`

---

## Work Type

**Pure feature** — новые MCP handler'ы, не модифицирующие существующее поведение P0+P1. Нет defect'а для воспроизведения; есть spec для реализации.

Task order для Pure Feature: `GREEN (test stubs) → CODE (impl, bottom-up) → GREEN (full tests) → GATE`.

> **Структурное замечание:** task-plan разбит на 3 секции по батчам реализации (см. ADR-7 в design). Каждая секция содержит 4–6 top-level задач. Финальный `GATE` — общий. Итого ~19 top-level задач в плане. Внутри батча задачи независимы; между батчами зависимости минимальны и явно отмечены.

---

## Test Style Source

**Test Style Source:** Tier 2 — adjacent test files

- Reference files:
  - `internal/handler/mock_client_test.go` — паттерн function-field mock
  - `internal/handler/diagnostics_test.go` — read-only handler tests
  - `internal/handler/nodes_test.go` — composite handler tests (`AddWorkerNode` polling)
  - `internal/handler/modules_test.go` — update-handler tests с unstructured
  - `internal/handler/config_test.go` — Secret-handler tests
- Key patterns:
  - Standard Go `testing` package, no external test framework
  - `mockClient` с function-field полями, дефолт = nil (возврат zero value / nil error)
  - Table-driven через `t.Run(name, func(t *testing.T) {...})`
  - Fixtures: `*unstructured.Unstructured` literals для CRD, `corev1.*` literals для core
  - Assertions: `errors.Is`, `strings.Contains` на error-сообщения, `reflect.DeepEqual` или поле-за-полем сравнение
  - Polling-тесты принимают реальный `time.Sleep` (без инжектируемого clock)
- **PBT unavailable** — using targeted unit tests as substitute (см. design §2.8 «Property-Based Tests» table)

---

## Commands

| Action   | Command            | Source        |
|----------|--------------------|---------------|
| Test     | `task test`        | Taskfile.yml  |
| Build    | `task build`       | Taskfile.yml  |
| Lint     | `task lint`        | Taskfile.yml  |
| Generate | `task generate`    | Taskfile.yml  |
| Integration | `task integration` | Taskfile.yml |

> `task generate` обязателен после каждого изменения `.proto`. Запускается **до** `task build` / `task test`, чтобы регенерированные `*.pb.go` и `*.mcp.go` попали в компиляцию.

---

## Coverage Matrix (REQ → Task → CP)

| Requirement | Task(s) | Correctness Property |
|-------------|---------|----------------------|
| REQ-1.1 | T-1, T-2, T-3 | CP-1 |
| REQ-1.2 | T-2, T-3 | CP-2 |
| REQ-1.3 | T-1, T-2, T-3 | CP-3 |
| REQ-1.4 | T-2, T-3 | CP-2 |
| REQ-1.5 | T-1, T-2, T-3 | CP-4 |
| REQ-1.6 | T-2, T-3 | CP-2 |
| REQ-2.1 | T-1, T-2, T-3 | CP-5 |
| REQ-2.2 | T-6, T-7, T-9 | CP-6 |
| REQ-2.3 | T-7, T-9 | CP-2 |
| REQ-2.4 | T-7, T-9 | CP-7 |
| REQ-3.1 | T-1, T-2, T-3 | CP-8 |
| REQ-3.2 | T-6, T-7, T-8 | CP-9 |
| REQ-3.3 | T-2, T-3, T-7, T-8 | CP-2 |
| REQ-3.4 | T-6, T-7, T-10 | CP-10, CP-11 |
| REQ-3.5 | T-7, T-10 | CP-12 |
| REQ-3.6 | T-7, T-10 | CP-13 |
| REQ-3.7 | T-6, T-7, T-8 | CP-14 |
| REQ-3.8 | T-7, T-8 | CP-2, CP-14 |
| REQ-3.9 | T-6, T-7, T-8 | CP-14 |
| REQ-3.10 | T-7, T-8 | CP-2, CP-14 |
| REQ-4.1 | T-1, T-2, T-3 | CP-15 |
| REQ-4.2 | T-2, T-3 | CP-2, CP-15 |
| REQ-4.3 | T-6, T-7, T-11 | CP-16 |
| REQ-4.4 | T-7, T-11 | CP-2 |
| REQ-5.1 | T-14, T-15, T-16 | CP-17 |
| REQ-5.2 | T-14, T-15, T-16 | CP-19 |
| REQ-5.3 | T-15, T-16 | CP-18 |
| REQ-5.4 | T-14, T-15, T-16 | CP-17 |
| REQ-5.5 | T-14, T-15, T-16 | CP-19 |
| REQ-5.6 | T-15, T-16 | CP-18 |
| REQ-6.1 | T-1, T-6, T-14, T-15 | CP-20 |
| REQ-6.2 | T-4, T-12, T-17 | CP-21 |
| REQ-6.3 | T-4, T-12, T-17 | CP-22 |
| REQ-6.4 | T-1, T-6, T-14 | CP-23 |
| REQ-6.5 | T-5, T-13, T-18, T-19 | CP-24 |
| REQ-6.6 | T-5, T-13, T-18, T-19 | CP-24 |

Все 32 требования покрыты ≥ 1 задачей.

---

# Batch 1 — Read-only (6 handlers)

> Группа requirement'ов: REQ-1.1–1.6, REQ-2.1, REQ-3.1, REQ-4.1–4.2.
> Handler'ы: `GetNodeEvents`, `GetStaticInstance`, `GetPodLogs`, `ListModules`, `CordonNode`, `GetStaticClusterConfiguration`.
> Зависимости: нет.

---

### T-1: Расширить proto-определения и k8s.Client для Batch 1

*_Requirements: 1.1, 1.3, 1.5, 2.1, 3.1, 4.1, 6.1, 6.4_*
*_Preservation: CP-20, CP-23_*
*_Complexity: standard_*

GOAL: Добавить новые RPC в существующие proto services и расширить `k8s.Client` для Batch 1 handler'ов.

CRITICAL: После каждого `.proto` изменения запускать `task generate` ДО `task build` / `task test`.
IMPORTANT: Не модифицировать существующие RPC и messages — только аддитивные изменения (см. ADR-6).
DO NOT: Реализовывать сами handler'ы в этом task. Только proto + k8s.Client signatures + GVR.

Subtasks:
- [ ] 1. В файле `proto/deckhouse/v1/diagnostics.proto`: добавить RPC `GetNodeEvents`, `GetStaticInstance`, `GetPodLogs` в `service DiagnosticsAPI` и соответствующие request/response messages согласно `design.md` §2.5. Аннотации `read_only_hint: true`. Запустить `task generate`.
- [ ] 2. В файле `proto/deckhouse/v1/modules.proto`: добавить RPC `ListModules` + `ListModulesResponse` + message `Module`. Аннотация `read_only_hint: true`. Запустить `task generate`.
- [ ] 3. В файле `proto/deckhouse/v1/nodes.proto`: добавить RPC `CordonNode` + `CordonNodeRequest` + `CordonNodeResponse`. Аннотация `destructive_hint: true`. Запустить `task generate`.
- [ ] 4. В файле `proto/deckhouse/v1/config.proto`: добавить RPC `GetStaticClusterConfiguration` + `GetStaticClusterConfigurationResponse`. Аннотация `read_only_hint: true`. Запустить `task generate`.
- [ ] 5. В файле `internal/k8s/client.go`: добавить GVR-константу `ModuleGVR` (group=deckhouse.io, version=v1alpha1, resource=modules) под существующими GVR.
- [ ] 6. В файле `internal/k8s/client.go`: добавить метод `ListModules(ctx context.Context) ([]unstructured.Unstructured, error)` в interface `Client` и реализацию на `*client` через `c.dynamic.Resource(ModuleGVR).List(...)`.

После всех subtasks: `task generate && task build && task lint` — должны пройти без ошибок.

---

### T-2: GREEN — написать unit-тесты для Batch 1 handler'ов

*_Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 2.1, 3.1, 3.3, 4.1, 4.2_*
*_Test_Style: Tier 2 — `internal/handler/diagnostics_test.go`, `modules_test.go`, `nodes_test.go`, `config_test.go`_*
*_Complexity: standard_*

GOAL: Написать unit-тесты для всех 6 read-only handler'ов **до** их реализации. На текущем коде тесты должны падать или не компилироваться.

CRITICAL: Тесты должны быть конкретными — указывать имена мок-полей, ожидаемые поля response, ожидаемые ошибки. См. design §2.8 «Unit Tests» как источник имён.
IMPORTANT: Если методы handler'а ещё не существуют, тесты не скомпилируются — это ожидаемо для GREEN-stub фазы. После реализации (T-3) они должны проходить.
DO NOT: Писать prod-код handler'ов в этом task. Только тесты + минимально необходимое расширение `mockClient`.

Subtasks:
- [ ] 1. В файле `internal/handler/mock_client_test.go`: добавить function-field `ListModulesFunc func(ctx context.Context) ([]unstructured.Unstructured, error)` в struct `mockClient` и его метод-обёртку. (`GetStaticInstance`, `ListNodeEvents`, `GetPodLogs`, `GetSecret`, `GetNode` уже существуют — не дублировать.)
- [ ] 2. В файле `internal/handler/diagnostics_test.go`: добавить тесты `TestGetNodeEvents_Happy`, `TestGetNodeEvents_NoEvents`, `TestGetNodeEvents_NotFound`, `TestGetStaticInstance_Happy`, `TestGetStaticInstance_NotFound`, `TestGetPodLogs_Happy`, `TestGetPodLogs_NotFound`. Соответствия CP — см. design §2.8.
- [ ] 3. В файле `internal/handler/modules_test.go`: добавить `TestListModules_Happy`, `TestListModules_Empty`. Fixture: 2 `*unstructured.Unstructured` с разными `spec.weight`/`spec.source`/`status.state`.
- [ ] 4. В файле `internal/handler/nodes_test.go`: добавить `TestCordonNode_Happy`, `TestCordonNode_AlreadyCordoned`, `TestCordonNode_NotFound`. Mock сначала `GetNodeFunc` (для previousState), затем `CordonNodeFunc`. См. ADR-1.
- [ ] 5. В файле `internal/handler/config_test.go`: добавить `TestGetStaticClusterConfiguration_Happy`, `TestGetStaticClusterConfiguration_KeyMissing`, `TestGetStaticClusterConfiguration_SecretMissing`. Fixture: Secret с двумя ключами `cluster-configuration.yaml` и `static-cluster-configuration.yaml`.
- [ ] 6. Запустить `task test ./internal/handler/...` — все новые тесты должны падать (методов handler'а нет) либо не компилироваться. Это ожидаемый GREEN-stub статус.

---

### T-3: CODE — реализовать Batch 1 handler'ы

*_Requirements: 1.1, 1.3, 1.5, 2.1, 3.1, 4.1_*
*_Preservation: CP-1, CP-2, CP-3, CP-4, CP-5, CP-8, CP-15, CP-24_*
*_Complexity: standard_*

GOAL: Реализовать 6 read-only handler-методов так, чтобы тесты из T-2 проходили, а существующие тесты P0+P1 (70 шт.) не регрессировали.

CRITICAL: Каждый subtask меняет только указанный файл.
IMPORTANT: После каждого subtask запускать `task test ./internal/handler/...` и проверять, что:
1. Соответствующие тесты из T-2 переходят в PASS.
2. Тесты P0+P1 остаются GREEN.
DO NOT: Изменять сигнатуру существующих методов `k8s.Client`. Использовать только новые методы + существующие.

Subtasks:
- [ ] 1. В файле `internal/handler/diagnostics.go`: реализовать `GetNodeEvents(ctx, req)` — вызов `client.ListNodeEvents(ctx, req.GetName())`, mapping `corev1.Event` → `pb.NodeEvent`, sort по `LastTimestamp`. Запустить `task test ./internal/handler -run TestGetNodeEvents`.
- [ ] 2. В файле `internal/handler/diagnostics.go`: реализовать `GetStaticInstance(ctx, req)` — вызов `client.GetStaticInstance(ctx, req.GetName())`, извлечение полей через `unstructured.NestedString`/`NestedMap`. Запустить `task test ./internal/handler -run TestGetStaticInstance`.
- [ ] 3. В файле `internal/handler/diagnostics.go`: реализовать `GetPodLogs(ctx, req)` — прямой проброс параметров (namespace, pod, container, tail, since) в `client.GetPodLogs`. Запустить `task test ./internal/handler -run TestGetPodLogs`.
- [ ] 4. В файле `internal/handler/modules.go`: реализовать `ListModules(ctx, req)` — вызов `client.ListModules(ctx)`, mapping каждого `unstructured.Unstructured` в `pb.Module` (name из metadata, weight/source из spec, state из status). Запустить `task test ./internal/handler -run TestListModules`.
- [ ] 5. В файле `internal/handler/nodes.go`: реализовать `CordonNode(ctx, req)` — `client.GetNode(name)` → `previousState := node.Spec.Unschedulable` → `client.CordonNode(name)` → return `previous_state`. Запустить `task test ./internal/handler -run TestCordonNode`.
- [ ] 6. В файле `internal/handler/config.go`: реализовать `GetStaticClusterConfiguration(ctx, req)` — `client.GetSecret("kube-system", "d8-cluster-configuration")` → достать ключ `static-cluster-configuration.yaml` → возврат как string. Запустить `task test ./internal/handler -run TestGetStaticClusterConfiguration`.

После всех subtasks: `task generate && task build && task lint && task test` — без ошибок.

---

### T-4: Расширить RBAC и регистрацию handler'ов для Batch 1

*_Requirements: 6.2, 6.3_*
*_Preservation: CP-21, CP-22_*
*_Complexity: mechanical_*

GOAL: Обновить deployment-манифесты RBAC и регистрацию новых handler-методов в server entrypoint.

IMPORTANT: Из 6 handler'ов Batch 1 новые RBAC-права нужны только для `ListModules` (CRD `modules`). Остальные используют уже-разрешённые ресурсы (`nodes`, `pods/log`, `events`, `staticinstances`, `secrets:get`).
NOTE: Регистрация generated `pb.Register*Tools` в `main.go` идёт для существующих сервисов — добавить вызов `pb.RegisterSourcesAPITools` НЕ требуется в Batch 1 (Sources идёт в Batch 3).

Subtasks:
- [ ] 1. В файле `deploy/rbac.yaml`: добавить правило `apiGroups: ["deckhouse.io"], resources: ["modules"], verbs: ["get", "list"]`.
- [ ] 2. В файле `cmd/deckhouse-mcp/main.go`: проверить, что новые RPC автоматически зарегистрированы через существующие `pb.RegisterDiagnosticsAPITools`, `pb.RegisterModulesAPITools`, `pb.RegisterNodesAPITools`, `pb.RegisterConfigAPITools` — после `task generate` они должны включить новые методы. Если требуется явное обновление вызова — обновить.
- [ ] 3. Запустить `task generate && task build` — убедиться, что main.go компилируется со всеми новыми handler-методами.

---

### T-5: GATE — Batch 1 verification

*_Requirements: 6.5, 6.6_*
*_Preservation: CP-24_*
*_Complexity: mechanical_*

GOAL: Подтвердить, что Batch 1 готов к merge: все новые тесты зелёные, P0+P1 не регрессирует, lint+generate проходят.

CRITICAL: Не переходить к Batch 2, пока этот GATE не зелёный.

Instructions:
1. Запустить `task generate` — exit 0, никаких изменений в generated files после re-run.
2. Запустить `task build` — exit 0.
3. Запустить `task lint` — exit 0.
4. Запустить `task test` — все тесты PASS. Счётчик passing-тестов ≥ 70 (P0+P1) + ≥ 17 (Batch 1).
5. Проверить покрытие: каждое из REQ-1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 2.1, 3.1, 3.3, 4.1, 4.2 имеет минимум 1 проходящий тест.
6. Если хоть один шаг fail — вернуться к соответствующей задаче в Batch 1 и исправить.

---

# Batch 2 — Writes (6 handlers)

> Группа requirement'ов: REQ-2.2–2.4, REQ-3.2, REQ-3.4–3.10, REQ-4.3–4.4.
> Handler'ы: `UpdateModuleSettings`, `DeleteSSHCredentials`, `UncordonNode`, `DrainNode`, `DeleteNodeGroup`, `UpdateKubernetesVersion`.
> Зависимости: Batch 1 (CordonNode уже реализован; используется в DrainNode опосредованно через `k8s.Client.CordonNode`).

---

### T-6: Расширить proto-определения и k8s.Client для Batch 2

*_Requirements: 2.2, 3.2, 3.4, 3.7, 3.9, 4.3, 6.1_*
*_Preservation: CP-20_*
*_Complexity: standard_*

GOAL: Добавить новые RPC и расширить `k8s.Client` 5 методами, нужными для writes-handler'ов Batch 2.

CRITICAL: `EvictPod` — новый метод, использующий `policy/v1` `Eviction` API. Не путать с существующим `DeletePod`.
IMPORTANT: `UpdateSecret` принимает весь `*corev1.Secret` (с ResourceVersion для optimistic concurrency).

Subtasks:
- [ ] 1. В файле `proto/deckhouse/v1/modules.proto`: добавить RPC `UpdateModuleSettings` + `UpdateModuleSettingsRequest` (поля: `string name`, `google.protobuf.Struct settings`) + `UpdateModuleSettingsResponse`. Аннотация `destructive_hint: false` (это update, не destructive). Запустить `task generate`.
- [ ] 2. В файле `proto/deckhouse/v1/nodes.proto`: добавить RPC `UncordonNode`, `DrainNode`, `DeleteSSHCredentials`, `DeleteNodeGroup` + соответствующие messages. Аннотации `destructive_hint: true` для delete и drain, `destructive_hint: false` для uncordon. Запустить `task generate`.
- [ ] 3. В файле `proto/deckhouse/v1/config.proto`: добавить RPC `UpdateKubernetesVersion` + `UpdateKubernetesVersionRequest` (`string version`) + `UpdateKubernetesVersionResponse` (`bool updated`, `string previous_version`). Аннотация `destructive_hint: true`. Запустить `task generate`.
- [ ] 4. В файле `internal/k8s/client.go`: добавить в interface `Client` 5 методов: `UncordonNode(ctx, name) error`, `EvictPod(ctx, namespace, name) error`, `UpdateSecret(ctx, secret) (*corev1.Secret, error)`, `DeleteSSHCredentials(ctx, name) error`, `DeleteNodeGroup(ctx, name) error`.
- [ ] 5. В файле `internal/k8s/client.go`: реализовать эти 5 методов на `*client`:
  - `UncordonNode`: `Get(name) → node.Spec.Unschedulable = false → Update`. Симметрично существующему `CordonNode`.
  - `EvictPod`: построить `policyv1.Eviction{ObjectMeta: {Namespace, Name}}` и вызвать `c.typed.CoreV1().Pods(namespace).EvictV1(ctx, eviction)`.
  - `UpdateSecret`: `c.typed.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{})`.
  - `DeleteSSHCredentials`, `DeleteNodeGroup`: `c.dynamic.Resource(GVR).Delete(ctx, name, metav1.DeleteOptions{})`.
- [ ] 6. Запустить `task generate && task build && task lint` — без ошибок.

---

### T-7: GREEN — написать unit-тесты для Batch 2 handler'ов

*_Requirements: 2.2, 2.3, 2.4, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7, 3.8, 3.9, 3.10, 4.3, 4.4_*
*_Test_Style: Tier 2 — `internal/handler/nodes_test.go` (AddWorkerNode polling pattern), `modules_test.go`, `config_test.go`_*
*_Complexity: complex_*

GOAL: Написать unit-тесты для 6 writes-handler'ов до их реализации. Тесты должны падать на текущем коде.

CRITICAL: `TestDrainNode_*` тесты используют polling 30s — каждый займёт ~30s minimum (как существующие `AddWorkerNode`-тесты).
IMPORTANT: Тесты на deep-merge (`TestUpdateModuleSettings_Happy`, `_NullRemoves`) должны охватывать 5 сценариев из CP-6 generator description.
DO NOT: Реализовывать handler'ы в этом task.

Subtasks:
- [ ] 1. В файле `internal/handler/mock_client_test.go`: добавить function-fields для 5 новых k8s.Client методов: `UncordonNodeFunc`, `EvictPodFunc`, `UpdateSecretFunc`, `DeleteSSHCredentialsFunc`, `DeleteNodeGroupFunc`. Дефолт-значения функций возвращают `nil` / zero. Добавить compile-time assertion `var _ k8s.Client = (*mockClient)(nil)` в новый тест `TestMockClient_ImplementsInterface`.
- [ ] 2. В файле `internal/handler/modules_test.go`: добавить `TestUpdateModuleSettings_Happy`, `TestUpdateModuleSettings_NullRemoves`, `TestUpdateModuleSettings_Empty`, `TestUpdateModuleSettings_NotFound`. Fixtures: ModuleConfig с nested `spec.settings = {a: 1, nested: {b: 2, c: 3}}`. Request scenarios см. design §2.8 «propUpdateModuleSettingsDeepMerge».
- [ ] 3. В файле `internal/handler/nodes_test.go`: добавить `TestUncordonNode_*` (3 теста, симметричны `CordonNode`), `TestDeleteSSHCredentials_Happy`, `TestDeleteSSHCredentials_NotFound`, `TestDeleteNodeGroup_Happy`, `TestDeleteNodeGroup_NotFound`. Mock `GetNodeFunc` для previousState.
- [ ] 4. В файле `internal/handler/nodes_test.go`: добавить `TestDrainNode_Happy`, `TestDrainNode_SkipsDaemonSet`, `TestDrainNode_SkipsMirror`, `TestDrainNode_CordonFails`, `TestDrainNode_PodAlreadyGone`. Использовать короткий `timeout_seconds=2` или fixture, где first round выселяет всех — иначе тесты будут полу-минутные.
- [ ] 5. В файле `internal/handler/nodes_test.go`: добавить `TestDrainNode_PDBBlocksThenSucceeds` и `TestDrainNode_Timeout`. Mock `EvictPodFunc` с counter, чтобы возвращать `apierrors.NewTooManyRequests(...)` на первых вызовах и success/timeout — на последующих. NOTE: эти тесты длятся ≥ 30s каждый из-за polling.
- [ ] 6. В файле `internal/handler/config_test.go`: добавить `TestUpdateKubernetesVersion_Happy`, `TestUpdateKubernetesVersion_SecretMissing`, `TestUpdateKubernetesVersion_KeyMissing`, `TestUpdateKubernetesVersion_RetryOnConflict`. Fixture: Secret с `cluster-configuration.yaml: "apiVersion: deckhouse.io/v1\nkind: ClusterConfiguration\nkubernetesVersion: \"1.28\"\nclusterDomain: cluster.local"`. Retry-тест: `UpdateSecretFunc` возвращает `apierrors.NewConflict` на 1-м вызове, success на 2-м.
- [ ] 7. Запустить `task test ./internal/handler/...` — новые тесты должны падать (методы handler'а не существуют) или не компилироваться.

---

### T-8: CODE — реализовать простые writes (Uncordon, Delete*)

*_Requirements: 3.2, 3.3, 3.7, 3.8, 3.9, 3.10_*
*_Preservation: CP-2, CP-9, CP-14, CP-24_*
*_Complexity: standard_*

GOAL: Реализовать 3 «простых» writes-handler'а: `UncordonNode`, `DeleteSSHCredentials`, `DeleteNodeGroup`. Они изоморфны существующему `CordonNode` (Batch 1) и `DeleteStaticInstance` (P1).

Subtasks:
- [ ] 1. В файле `internal/handler/nodes.go`: реализовать `UncordonNode(ctx, req)` — `client.GetNode(name)` → `previousState := node.Spec.Unschedulable` → `client.UncordonNode(name)` → return `previous_state`. Запустить `task test ./internal/handler -run TestUncordonNode`.
- [ ] 2. В файле `internal/handler/nodes.go`: реализовать `DeleteSSHCredentials(ctx, req)` — простой проброс в `client.DeleteSSHCredentials(req.GetName())`, обёрнутая ошибка `fmt.Errorf("deleting SSHCredentials %s: %w", name, err)`. Запустить `task test ./internal/handler -run TestDeleteSSHCredentials`.
- [ ] 3. В файле `internal/handler/nodes.go`: реализовать `DeleteNodeGroup(ctx, req)` — аналогично, через `client.DeleteNodeGroup`. Запустить `task test ./internal/handler -run TestDeleteNodeGroup`.

После всех subtasks: `task test ./internal/handler/...` — все Uncordon/Delete тесты PASS, P0+P1 GREEN.

---

### T-9: CODE — реализовать `UpdateModuleSettings` (deep merge RFC 7396)

*_Requirements: 2.2, 2.3, 2.4_*
*_Preservation: CP-2, CP-6, CP-7, CP-24_*
*_Complexity: complex_*

GOAL: Реализовать deep-merge update для `spec.settings` через JSON Merge Patch (RFC 7396), как обосновано в ADR-2.

CRITICAL: Использовать существующую indirect-зависимость `gopkg.in/evanphx/json-patch.v4` (она уже в `go.sum` через `client-go`). НЕ добавлять новых dependency.
IMPORTANT: Empty settings (`len(req.Settings.Fields) == 0`) должен возвращать ошибку валидации БЕЗ вызова `GetModuleConfig` — для прохождения `TestUpdateModuleSettings_Empty`.

Subtasks:
- [ ] 1. В файле `internal/handler/modules.go`: добавить локальную error-переменную `var errEmptyModuleSettings = errors.New("settings cannot be empty")` под существующими error-переменными.
- [ ] 2. В файле `internal/handler/modules.go`: реализовать `UpdateModuleSettings(ctx, req)` — pre-call validation (`if len(req.Settings.Fields) == 0 { return nil, errEmptyModuleSettings }`), затем `GetModuleConfig(req.Name)`, deep-merge с использованием `jsonpatch.MergePatch(existing, patch)` (RFC 7396), assignment merged value в `mc.Object.spec.settings`, вызов `UpdateModuleConfig(mc)`. Запустить `task test ./internal/handler -run TestUpdateModuleSettings`.
- [ ] 3. После passing tests: запустить `task lint` — `easyp lint` не должен выдавать warnings на новую proto-аннотацию `google.protobuf.Struct settings`.

---

### T-10: CODE — реализовать `DrainNode` (composite, Eviction API + polling)

*_Requirements: 3.4, 3.5, 3.6_*
*_Preservation: CP-2, CP-10, CP-11, CP-12, CP-13, CP-24_*
*_Complexity: complex_*

GOAL: Реализовать composite handler `DrainNode`: cordon → list pods on node → eviction loop с polling 30s до timeout. Согласно ADR-3, ADR-4.

CRITICAL: Использовать НОВЫЙ приватный helper `evictPodsWithPDB`, НЕ переиспользовать существующий приватный `drainNode` (он используется `RemoveNode` и должен остаться без изменений — см. ADR-4).
IMPORTANT: Helper должен исключать DaemonSet-pods (через `isDaemonSetPod` — уже есть) и mirror-pods (`pod.Annotations["kubernetes.io/config.mirror"] != ""`).
IMPORTANT: Обработка ошибок `EvictPod`:
- `apierrors.IsTooManyRequests` (PDB) → pod остаётся в очереди для следующего раунда
- `apierrors.IsNotFound` → pod уже исчез, считаем evicted
- остальное → pod в `failed_pods`, продолжаем polling
DO NOT: Изменять существующий `drainNode` helper и `RemoveNode`.

Subtasks:
- [ ] 1. В файле `internal/handler/nodes.go`: добавить константу `const drainTimeoutSeconds = 300` рядом с существующими константами.
- [ ] 2. В файле `internal/handler/nodes.go`: добавить приватную функцию `isMirrorPod(pod *corev1.Pod) bool` — проверяет `pod.Annotations["kubernetes.io/config.mirror"] != ""`. Расположить рядом с `isDaemonSetPod`.
- [ ] 3. В файле `internal/handler/nodes.go`: реализовать helper `(h *NodesHandler) evictPodsWithPDB(ctx context.Context, nodeName string, deadline time.Time) (evictedCount int32, failedPods []string, timedOut bool, err error)`. Алгоритм:
  - В loop: `ListPods("")` → отфильтровать по `nodeName` + non-DS + non-mirror.
  - Для каждого pod: `client.EvictPod(namespace, name)`. Классифицировать err через `apierrors.IsTooManyRequests`, `IsNotFound`, прочее.
  - Если все evicted/notfound — return success.
  - Иначе: `if time.Now().After(deadline) { timedOut = true; break }`.
  - `select { case <-ctx.Done(): return ctx.Err(); case <-time.After(pollInterval): }`.
- [ ] 4. В файле `internal/handler/nodes.go`: реализовать MCP-метод `DrainNode(ctx, req)`. Шаги:
  - Resolve `timeout` (`req.TimeoutSeconds`, default = `drainTimeoutSeconds`).
  - `client.CordonNode(req.Name)`. На ошибку — wrap и return.
  - Вычислить `deadline := time.Now().Add(timeout)`.
  - Вызвать `evictPodsWithPDB(ctx, req.Name, deadline)`.
  - Сформировать `DrainNodeResponse{Cordoned: true, EvictedCount, FailedPods, TimedOut, Elapsed}`.
- [ ] 5. Запустить `task test ./internal/handler -run TestDrainNode` — все 7 DrainNode-тестов PASS. Время выполнения: ≥ 30s × 2 polling-теста ≈ 1 минута.

После всех subtasks: `task generate && task build && task lint && task test`.

---

### T-11: CODE — реализовать `UpdateKubernetesVersion` (YAML round-trip + retry)

*_Requirements: 4.3, 4.4_*
*_Preservation: CP-2, CP-16, CP-24_*
*_Complexity: complex_*

GOAL: Реализовать read-modify-write Secret-а с YAML парсингом через `sigs.k8s.io/yaml` (ADR-5), retry на `IsConflict` до 3 раз.

CRITICAL: Использовать `sigs.k8s.io/yaml` (уже в indirect deps), НЕ `gopkg.in/yaml.v3` и НЕ `go.yaml.in/yaml/*`.
IMPORTANT: Имя ключа в Secret — `cluster-configuration.yaml` (НЕ `static-cluster-configuration.yaml` — это другой ключ для GetStaticClusterConfiguration в Batch 1).

Subtasks:
- [ ] 1. В файле `internal/handler/config.go`: добавить импорт `"sigs.k8s.io/yaml"`.
- [ ] 2. В файле `internal/handler/config.go`: реализовать `UpdateKubernetesVersion(ctx, req)`. Алгоритм:
  - Loop до 3 попыток на `apierrors.IsConflict`:
    - `secret := client.GetSecret("kube-system", "d8-cluster-configuration")`.
    - `data, ok := secret.Data["cluster-configuration.yaml"]; if !ok { return error("key not found") }`.
    - `var cfg map[string]any; yaml.Unmarshal(data, &cfg)`. Сохранить `previousVersion := cfg["kubernetesVersion"]`.
    - `cfg["kubernetesVersion"] = req.GetVersion()`.
    - `newData, _ := yaml.Marshal(cfg)`. `secret.Data["cluster-configuration.yaml"] = newData`.
    - `_, err := client.UpdateSecret(secret)`. Если `apierrors.IsConflict(err)` — повторить loop.
  - Return `UpdateKubernetesVersionResponse{Updated: true, PreviousVersion: prev.(string)}`.
- [ ] 3. Запустить `task test ./internal/handler -run TestUpdateKubernetesVersion` — все 4 теста PASS.

---

### T-12: Расширить RBAC и mock для Batch 2

*_Requirements: 6.2, 6.3_*
*_Preservation: CP-21, CP-22_*
*_Complexity: mechanical_*

GOAL: Добавить RBAC-права для новых K8s операций Batch 2.

Subtasks:
- [ ] 1. В файле `deploy/rbac.yaml`: добавить правила:
  - `apiGroups: [""], resources: ["pods/eviction"], verbs: ["create"]` — для DrainNode.
  - `apiGroups: [""], resources: ["secrets"], verbs: ["update"]` в существующее правило (ограничение `resourceNames: ["d8-cluster-configuration"]` опционально).
  - `apiGroups: ["deckhouse.io"], resources: ["sshcredentials"], verbs: ["delete"]` — добавить `delete` к существующим verb-ам.
  - `apiGroups: ["deckhouse.io"], resources: ["nodegroups"], verbs: ["delete"]` — добавить `delete`.
  - `apiGroups: ["deckhouse.io"], resources: ["moduleconfigs"], verbs: ["update"]` — для UpdateModuleSettings (если ещё не разрешён).
- [ ] 2. В файле `internal/handler/mock_client_test.go`: убедиться, что все function-fields Batch 2 добавлены (см. T-7 subtask 1) и compile-time assertion `var _ k8s.Client = (*mockClient)(nil)` проходит.
- [ ] 3. Запустить `task build && task test ./internal/handler` — без ошибок.

---

### T-13: GATE — Batch 2 verification

*_Requirements: 6.5, 6.6_*
*_Preservation: CP-24_*
*_Complexity: mechanical_*

GOAL: Подтвердить, что Batch 2 готов к merge.

Instructions:
1. `task generate` — exit 0, no diff после re-run.
2. `task build` — exit 0.
3. `task lint` — exit 0.
4. `task test` — все тесты PASS. Счётчик: ≥ 70 (P0+P1) + ≥ 17 (Batch 1) + ≥ 18 (Batch 2) ≈ ≥ 105.
5. Проверить покрытие REQ-2.2–2.4, 3.2, 3.4–3.10, 4.3–4.4 — каждое имеет минимум 1 passing test.
6. Если fail — вернуться к Batch 2 задачам.

---

# Batch 3 — Sources (4 handlers, новый домен)

> Группа requirement'ов: REQ-5.1–5.6.
> Handler'ы: `ListModuleSources`, `CreateModuleSource`, `ListModuleUpdatePolicies`, `CreateModuleUpdatePolicy`.
> Зависимости: нет (полностью изолированный домен).

---

### T-14: Расширить proto-определения и k8s.Client для Batch 3

*_Requirements: 5.1, 5.2, 5.4, 5.5, 6.1, 6.4_*
*_Preservation: CP-20, CP-23_*
*_Complexity: standard_*

GOAL: Добавить 4 RPC в (пустой stub) `SourcesAPI`, 2 GVR-константы, 4 новых k8s.Client метода.

Subtasks:
- [ ] 1. В файле `proto/deckhouse/v1/sources.proto`: добавить service `SourcesAPI` с 4 RPC: `ListModuleSources`, `CreateModuleSource`, `ListModuleUpdatePolicies`, `CreateModuleUpdatePolicy`. Соответствующие request/response messages + helper messages `ModuleSource`, `ModuleUpdatePolicy` согласно design §2.5. Read-only хинты на List*, destructive_hint=false на Create*. Запустить `task generate`.
- [ ] 2. В файле `internal/k8s/client.go`: добавить GVR-константы `ModuleSourceGVR` (deckhouse.io/v1alpha1/modulesources) и `ModuleUpdatePolicyGVR` (deckhouse.io/v1alpha1/moduleupdatepolicies) под существующими GVR.
- [ ] 3. В файле `internal/k8s/client.go`: добавить 4 метода в interface `Client`: `ListModuleSources`, `CreateModuleSource`, `ListModuleUpdatePolicies`, `CreateModuleUpdatePolicy` (сигнатуры — см. design §2.3).
- [ ] 4. В файле `internal/k8s/client.go`: реализовать эти 4 метода через `c.dynamic.Resource(GVR).List/Create(...)`. Шаблоны — аналогичны существующим `ListModuleConfigs` и `CreateStaticInstance`.
- [ ] 5. Запустить `task generate && task build && task lint` — без ошибок.

---

### T-15: GREEN — написать unit-тесты для Sources handler'а

*_Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6_*
*_Test_Style: Tier 2 — `internal/handler/modules_test.go` (паттерн dynamic CRD list/create), `nodes_test.go` (CreateStaticInstance)_*
*_Complexity: standard_*

GOAL: Написать unit-тесты для 4 sources-методов до их реализации.

Subtasks:
- [ ] 1. В файле `internal/handler/mock_client_test.go`: добавить function-fields `ListModuleSourcesFunc`, `CreateModuleSourceFunc`, `ListModuleUpdatePoliciesFunc`, `CreateModuleUpdatePolicyFunc` со всеми их соответствующими методами-обёртками.
- [ ] 2. Создать новый файл `internal/handler/sources_test.go` с тестами `TestListModuleSources_Empty`, `TestListModuleSources_Happy`, `TestCreateModuleSource_Happy`, `TestCreateModuleSource_AlreadyExists`. Фикстуры: `*unstructured.Unstructured` с apiVersion=deckhouse.io/v1alpha1, kind=ModuleSource, spec.registry, status.
- [ ] 3. В том же файле: добавить тесты `TestListModuleUpdatePolicies_Empty`, `TestListModuleUpdatePolicies_Happy`, `TestCreateModuleUpdatePolicy_Happy`, `TestCreateModuleUpdatePolicy_AlreadyExists`. Mock возвращает `apierrors.NewAlreadyExists(...)` для duplicate-тестов.
- [ ] 4. Запустить `task test ./internal/handler -run "Test(List|Create)Module(Source|UpdatePolicy)"` — все тесты должны падать (sources.go ещё не создан).

---

### T-16: CODE — реализовать `SourcesHandler` (4 метода)

*_Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6_*
*_Preservation: CP-17, CP-18, CP-19, CP-24_*
*_Complexity: standard_*

GOAL: Создать новый handler-файл `sources.go` и реализовать 4 метода.

Subtasks:
- [ ] 1. Создать новый файл `internal/handler/sources.go` с пакетом `handler`, импортами, struct `SourcesHandler` и конструктором `NewSourcesHandler(client k8s.Client) *SourcesHandler`. Шаблон — см. `releases.go` (P1).
- [ ] 2. В том же файле реализовать `ListModuleSources(ctx, req)` — вызов `client.ListModuleSources(ctx)`, mapping каждого `unstructured.Unstructured` в `pb.ModuleSource{Name, Registry, Status}`. Запустить `task test ./internal/handler -run TestListModuleSources`.
- [ ] 3. В том же файле реализовать `CreateModuleSource(ctx, req)` — построение `unstructured.Unstructured{apiVersion: "deckhouse.io/v1alpha1", kind: "ModuleSource", metadata: {name}, spec: {registry: req.Registry}}`, вызов `client.CreateModuleSource`. Запустить `task test ./internal/handler -run TestCreateModuleSource`.
- [ ] 4. В том же файле реализовать `ListModuleUpdatePolicies(ctx, req)` и `CreateModuleUpdatePolicy(ctx, req)` — аналогично, kind=`ModuleUpdatePolicy`, spec.updateMode. Запустить `task test ./internal/handler -run "Test(List|Create)ModuleUpdatePolicy"`.
- [ ] 5. В файле `cmd/deckhouse-mcp/main.go`: добавить инстанцирование `sourcesHandler := handler.NewSourcesHandler(client)` и регистрацию `pb.RegisterSourcesAPITools(server, sourcesHandler)`. Запустить `task build`.

После всех subtasks: `task generate && task build && task lint && task test`.

---

### T-17: Расширить RBAC и integration CRDs для Batch 3

*_Requirements: 6.2, 6.3_*
*_Preservation: CP-22_*
*_Complexity: mechanical_*

GOAL: Добавить RBAC и CRD-определения для integration-тестов.

Subtasks:
- [ ] 1. В файле `deploy/rbac.yaml`: добавить правила:
  - `apiGroups: ["deckhouse.io"], resources: ["modulesources"], verbs: ["get", "list", "create"]`.
  - `apiGroups: ["deckhouse.io"], resources: ["moduleupdatepolicies"], verbs: ["get", "list", "create"]`.
- [ ] 2. В файле `tests/integration/crds.yaml`: добавить минимальные CRD-определения для `ModuleSource` (deckhouse.io/v1alpha1, spec: {registry: string}) и `ModuleUpdatePolicy` (deckhouse.io/v1alpha1, spec: {updateMode: string}). Также добавить `Module` (deckhouse.io/v1alpha1, spec: {weight, source}, status: {state}) для тестирования B7 (ListModules) из Batch 1.
- [ ] 3. Запустить `task build && task test` — без ошибок.

---

### T-18: GATE — Batch 3 verification

*_Requirements: 6.5, 6.6_*
*_Preservation: CP-24_*
*_Complexity: mechanical_*

GOAL: Подтвердить, что Batch 3 готов к merge.

Instructions:
1. `task generate && task build && task lint && task test` — exit 0.
2. Счётчик тестов: ≥ 70 (P0+P1) + ≥ 17 (B1) + ≥ 18 (B2) + ≥ 8 (B3) ≈ ≥ 113.
3. Проверить покрытие REQ-5.1–5.6.
4. `main.go` регистрирует `pb.RegisterSourcesAPITools` — sources MCP tools доступны через `tools/list`.

---

# Final

### T-19: GATE — full feature verification

*_Requirements: ALL_*
*_Preservation: CP-24_*
*_Complexity: standard_*

CRITICAL: Этот task — последний в плане. Не помечать complete, пока ВСЕ предыдущие задачи не зелёные.

GOAL: Финальная проверка всей фичи P2: все 16 handler'ов работают, все 32 REQ покрыты passing-тестами, integration-test в Kind зелёный.

Instructions:
1. Запустить `task generate` — exit 0, no diff в `*.pb.go` / `*.mcp.go`.
2. Запустить `task build` — exit 0.
3. Запустить `task lint` — exit 0.
4. Запустить `task test` — все тесты PASS (≥ 113 тестов суммарно). Время: ~2 минуты (включая ~30s × 2 DrainNode polling + ~30s × 2 AddWorkerNode polling).
5. Запустить `task integration` — Kind-кластер поднимается, CRDs применяются из `tests/integration/crds.yaml`, MCP server отвечает на `tools/list` с полным каталогом (≥ 39 tools = 23 P0+P1 + 16 P2).
6. Проверить Coverage Matrix: каждое из REQ-1.1–1.6, 2.1–2.4, 3.1–3.10, 4.1–4.4, 5.1–5.6, 6.1–6.6 покрыто passing-тестом.
7. Manual RBAC audit (CP-22): для каждого из 16 новых handler'ов вручную проверить, что все его K8s операции разрешены в `deploy/rbac.yaml`. Документировать audit в commit-message финального merge-commit.
8. Обновить `CHANGELOG.md` секцией «## P2 — Advanced Management» со списком 16 новых handler'ов.
9. Если любой шаг fail — вернуться к соответствующему GATE батча и исправить.

---

## Quality Control

- ✅ Coverage matrix присутствует и полная (36 строк, все REQ покрыты)
- ✅ Work type классифицирован: **Pure feature**
- ✅ Task order: GREEN test stubs → CODE → GREEN full → GATE per batch + final GATE
- ✅ Каждый task имеет `*_Requirements:_*`
- ✅ Каждый CODE-task имеет `*_Preservation:_*`
- ✅ T-19 — финальный GATE
- ✅ Test Style Source указан (Tier 2)
- ✅ Commands block содержит реальные команды (`task test`, `task build`, `task lint`, `task generate`, `task integration`)
- ✅ Каждый top-level task имеет `*_Complexity:_*`
- ✅ Subtask'и атомарны, каждый — один файл
- ✅ Code generation учитывается: `task generate` запускается ДО `task build`/`task test` после каждого `.proto` изменения
- ✅ Артефакт регистрируется через `pipeline.sh artifact`
