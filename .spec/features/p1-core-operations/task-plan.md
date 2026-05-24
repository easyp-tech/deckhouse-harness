# Task Plan: P1 Core Operations

**Feature**: p1-core-operations  
**Phase**: [4/6] Task Plan  
**Work Type**: Pure feature — 13 новых MCP-инструментов, предшествующей реализации нет

---

## Преамбула

**Test Style Source:** Tier 2
- Reference files: `internal/handler/diagnostics_test.go`, `internal/handler/nodes_test.go`
- Паттерн: стандартный `testing.T`, table-driven тесты (`for _, tc := range []struct{...}{...}`), `mockClient` с function fields (nil = not called / not expected), вспомогательные `unstructured.Unstructured` builder-функции, имена `TestXxxHandler_Method_scenario`

**Commands:**
| Action   | Command           | Source       |
|----------|-------------------|--------------|
| Test     | `task test`       | Taskfile.yml |
| Build    | `task build`      | Taskfile.yml |
| Lint     | `task lint`       | Taskfile.yml |
| Generate | `task generate`   | Taskfile.yml |

---

## Матрица покрытия

| Requirement | Task(s) | Correctness Property |
|-------------|---------|----------------------|
| REQ-1.1     | T-3, T-4 | CP-10 (read-only, нет мутаций) |
| REQ-1.2     | T-3, T-4 | CP-9 (not found = явная ошибка) |
| REQ-1.3     | T-3, T-4 | — |
| REQ-1.4     | T-3, T-4 | — |
| REQ-1.5     | T-3, T-4 | — |
| REQ-1.6     | T-3, T-4 | CP-9 |
| REQ-1.7     | T-3, T-4 | — |
| REQ-1.8     | T-3, T-4 | CP-10 |
| REQ-1.9     | T-3, T-4 | — |
| REQ-1.10    | T-3, T-4 | — |
| REQ-1.11    | T-3, T-4 | — |
| REQ-1.12    | T-3, T-4 | CP-9 |
| REQ-2.1     | T-3, T-4 | — |
| REQ-2.2     | T-3, T-4 | CP-9 |
| REQ-2.3     | T-3, T-5 | CP-2, CP-8 |
| REQ-2.4     | T-3, T-5 | CP-8 (идемпотентность) |
| REQ-2.5     | T-3, T-5 | CP-9 |
| REQ-2.6     | T-3, T-5 | CP-2 |
| REQ-2.7     | T-3, T-5 | CP-9 |
| REQ-3.1     | T-3, T-4 | — |
| REQ-3.2     | T-3, T-4 | CP-9 |
| REQ-3.3     | T-3, T-5 | CP-7 (идемпотентность ApproveRelease) |
| REQ-3.4     | T-3, T-5 | CP-9 |
| REQ-4.1     | T-3, T-5 | CP-9 |
| REQ-4.2     | T-3, T-5 | CP-9 |
| REQ-4.3     | T-3, T-6 | CP-1 (нет парциального состояния) |
| REQ-4.4     | T-3, T-6 | CP-1 |
| REQ-4.5     | T-3, T-6 | CP-1 |
| REQ-4.6     | T-3, T-6 | — |
| REQ-4.7     | T-3, T-6 | CP-1 |
| REQ-4.8     | T-3, T-5 | — |
| REQ-4.9     | T-3, T-5 | — |
| REQ-4.10    | T-3, T-5 | CP-9 |
| REQ-4.11    | T-3, T-6 | CP-5 (polling всегда завершается) |
| REQ-4.12    | T-3, T-6 | CP-4, CP-9 |
| REQ-4.13    | T-3, T-6 | CP-5, CP-6 (timeout репортится) |
| REQ-4.14    | T-3, T-6 | — |
| REQ-5.1     | T-3, T-6 | CP-3, CP-11 |
| REQ-5.2     | T-3, T-6 | CP-9 |
| REQ-5.3     | T-3, T-6 | — |
| REQ-6.1     | T-1 | — |
| REQ-6.2     | T-1 | — |
| REQ-6.3     | T-3 | — |
| REQ-6.4     | T-7 | — |
| REQ-6.5     | T-2 | — |

---

## Correctness Properties (из Design §2.6)

| ID | Категория | Содержание |
|----|-----------|------------|
| CP-1  | Safety | Никакой мутации в RemoveNode до успешного GetNode/GetStaticInstance |
| CP-2  | Safety | EnableModule/DisableModule всегда читает current state перед записью |
| CP-3  | Safety | ClusterConfiguration возвращается verbatim, без маскирования |
| CP-4  | Safety | WaitNodeReady возвращает ошибку при первой же неудаче GetStaticInstance |
| CP-5  | Liveness | Polling (WaitNodeReady, AddWorkerNode) всегда завершается |
| CP-6  | Liveness | Таймаут возвращает `timedOut: true` + последнюю фазу (без ошибки) |
| CP-7  | Idempotency | ApproveRelease: повторный вызов = no-op, `previousApproved: true` |
| CP-8  | Idempotency | EnableModule с уже включённым модулем = `success: true, previousState: true` |
| CP-9  | Isolation | Ресурс не найден → явная ошибка с именем, не тихий success |
| CP-10 | Isolation | GetDeckhouseLogs читает логи, не меняет состояние кластера |
| CP-11 | Isolation | GetClusterConfiguration выполняет только GET на Secret |

---

## T-1 (CODE): Инфраструктура — k8s.Client + mockClient

*_Requirements: REQ-6.1, REQ-6.2_*  
*_Preservation: Существующие 9 методов Client (P0) не меняются; все существующие тесты проходят_*

GOAL: Расширить `k8s.Client` интерфейс и его реализацию 11 новыми методами, необходимыми для всех P1 хендлеров. Это фундамент: без него ни один P1 хендлер не скомпилируется.

IMPORTANT: Реализации методов добавляются в тот же файл `internal/k8s/client.go`, следуя существующему паттерну (typed client для core, dynamic для CRDs).

Subtasks:
- [ ] 1. Добавить 11 сигнатур в интерфейс `Client` в файл `internal/k8s/client.go`
- [ ] 2. Реализовать typed-методы (`GetNode`, `CordonNode`, `GetPodLogs`, `GetSecret`) в `internal/k8s/client.go`
- [ ] 3. Реализовать dynamic-методы (`GetNodeGroup`, `CreateNodeGroup`, `GetModuleConfig`, `UpdateModuleConfig`, `GetDeckhouseRelease`, `PatchDeckhouseRelease`, `DeleteStaticInstance`) в `internal/k8s/client.go`
- [ ] 4. Добавить 11 function fields в `mockClient` в `internal/handler/mock_client_test.go` с nil-check паттерном

После всех subtasks: `task build` — сборка должна пройти без ошибок.

---

## T-2 (CODE): Proto изменения + генерация кода

*_Requirements: REQ-6.5_*  
*_Preservation: Существующие proto RPCs (P0) не изменяются; generated code для P0 handlers не ломается_*

GOAL: Добавить 13 новых RPC во все 5 proto файлов и применить `task generate` для получения `*.pb.go` + `*.mcp.go`. Для `config.proto` это первый реальный RPC — `config.mcp.go` будет создан впервые.

IMPORTANT: Следовать MCP annotation conventions из Design §2.10: `read_only_hint: true` для read-only, `destructive_hint: true` для write/delete, `idempotent_hint: true` для Enable/Disable/Approve, field descriptions и `(mcp.options.v1.field)` constraints на всех входных полях.  
IMPORTANT: Enum zero-value `*_UNSPECIFIED = 0` помечать `(mcp.options.v1.enum_value) = { hidden: true }` если добавляются новые enums.

Subtasks:
- [ ] 1. Добавить RPCs `GetNode`, `GetNodeGroup`, `GetDeckhouseLogs` с сообщениями в `proto/deckhouse/v1/diagnostics.proto`
- [ ] 2. Добавить RPCs `GetModuleConfig`, `EnableModule`, `DisableModule` с сообщениями в `proto/deckhouse/v1/modules.proto`
- [ ] 3. Добавить RPCs `GetDeckhouseRelease`, `ApproveRelease` с сообщениями в `proto/deckhouse/v1/releases.proto`
- [ ] 4. Добавить RPCs `DeleteStaticInstance`, `RemoveNode`, `CreateNodeGroup`, `WaitNodeReady` с сообщениями в `proto/deckhouse/v1/nodes.proto`
- [ ] 5. Добавить `ConfigAPI` service с RPC `GetClusterConfiguration` в `proto/deckhouse/v1/config.proto`
- [ ] 6. Выполнить `task generate`, затем `task build` — оба должны пройти без ошибок

---

## T-3 (GREEN): Тесты ожидаемого поведения для всех 13 хендлеров

*_Requirements: REQ-1.1–1.12, REQ-2.1–2.7, REQ-3.1–3.4, REQ-4.1–4.14, REQ-5.1–5.3, REQ-6.3_*

GOAL: Написать unit-тесты для всех 13 новых хендлеров до их реализации. Тесты определяют ожидаемое поведение и компилируются, но падают (FAIL) до появления реализации.

IMPORTANT: Следовать Test Style Source (Tier 2): table-driven тесты, `mockClient` struct с function fields, имена `TestXxxHandler_Method_scenario`.  
IMPORTANT: Каждый хендлер должен иметь минимум 2 теста: happy path + not found/error case.  
CRITICAL: Тесты должны компилироваться (`task build`) и падать (`task test`) — именно в таком порядке проверять.  
DO NOT: Писать код реализации хендлеров в этом задании.

*_Test_Style:_* Tier 2 — `internal/handler/diagnostics_test.go`, `internal/handler/nodes_test.go`

Subtasks:
- [ ] 1. Добавить тесты `GetNode` (found+StaticInstance, found без SI, not found), `GetNodeGroup` (found с нодами, not found), `GetDeckhouseLogs` (success, pod not found, grep filter) в `internal/handler/diagnostics_test.go`
- [ ] 2. Добавить тесты `GetModuleConfig` (found с settings, not found), `EnableModule` (was disabled, already enabled, not found), `DisableModule` (was enabled, not found) в `internal/handler/modules_test.go`
- [ ] 3. Добавить тесты `GetDeckhouseRelease` (found+approved, found+pending, not found), `ApproveRelease` (success, already approved, not found) в `internal/handler/releases_test.go`
- [ ] 4. Добавить тесты `DeleteStaticInstance` (success, not found), `RemoveNode` (full flow drain=true, drain=false, no SI error), `CreateNodeGroup` (success, already exists), `WaitNodeReady` (reaches Running, times out) в `internal/handler/nodes_test.go`
- [ ] 5. Создать `internal/handler/config_test.go` с тестами `GetClusterConfiguration` (success, secret not found, invalid YAML)

После всех subtasks:
- `task build` → должна пройти без ошибок  
- `task test` → падают только тесты новых хендлеров (компиляция ОК)

---

## T-4 (CODE): G1 — Read-only GET хендлеры

*_Requirements: REQ-1.1–1.7, REQ-1.8–1.12, REQ-2.1–2.2, REQ-3.1–3.2_*  
*_Preservation: CP-10 (GetDeckhouseLogs read-only), CP-9 (not found = явная ошибка), CP-11 (нет мутаций). Существующие P0 хендлеры DiagnosticsHandler, ModulesHandler, ReleasesHandler работают без изменений._*

GOAL: Реализовать `GetNode`, `GetNodeGroup`, `GetDeckhouseLogs` в DiagnosticsHandler; `GetModuleConfig` в ModulesHandler; `GetDeckhouseRelease` в ReleasesHandler.

IMPORTANT: После каждого subtask запускать `task test` — тесты соответствующего хендлера должны стать GREEN; P0-тесты не должны регрессировать.  
IMPORTANT: `GetDeckhouseLogs` применяет grep-фильтрацию на стороне Go (strings.Contains), не передаёт grep в K8s API.  
IMPORTANT: `GetNodeGroup` включает список нод через отдельный вызов `client.ListNodes` + фильтрацию по label `node.deckhouse.io/group=<name>`.  
DO NOT: Создавать новые файлы — всё добавлять в существующие handler-файлы.

Subtasks:
- [ ] 1. Реализовать `GetNode` (включая optional StaticInstance phase lookup) и `GetNodeGroup` в `internal/handler/diagnostics.go` — `task test ./internal/handler/`
- [ ] 2. Реализовать `GetDeckhouseLogs` (find deckhouse pod → get logs → apply grep) в `internal/handler/diagnostics.go` — `task test ./internal/handler/`
- [ ] 3. Реализовать `GetModuleConfig` в `internal/handler/modules.go` — `task test ./internal/handler/`
- [ ] 4. Реализовать `GetDeckhouseRelease` в `internal/handler/releases.go` — `task test ./internal/handler/`

После всех subtasks: `task build` + `task lint`

---

## T-5 (CODE): G3 — Simple write хендлеры

*_Requirements: REQ-2.3–2.7, REQ-3.3–3.4, REQ-4.1–4.2, REQ-4.8–4.10_*  
*_Preservation: CP-2 (Get-before-write), CP-7 (ApproveRelease идемпотентен), CP-8 (EnableModule идемпотентен), CP-9 (not found error). Существующие P0 хендлеры и P0-тесты не регрессируют._*

GOAL: Реализовать `EnableModule`, `DisableModule` в ModulesHandler; `ApproveRelease` в ReleasesHandler; `DeleteStaticInstance`, `CreateNodeGroup` в NodesHandler.

IMPORTANT: `EnableModule`/`DisableModule` — паттерн Get → modify unstructured spec.enabled → full Update (ADR-3).  
IMPORTANT: `ApproveRelease` — merge patch только `metadata.annotations` (ADR-4).  
IMPORTANT: Аргументом `PatchDeckhouseRelease` передаётся `types.MergePatchType` ключ и JSON `{"metadata":{"annotations":{"release.deckhouse.io/approved":"true"}}}`.

*_Test_Style:_* Tier 2 — `internal/handler/modules_test.go`, `internal/handler/releases_test.go`

Subtasks:
- [ ] 1. Реализовать `EnableModule` и `DisableModule` в `internal/handler/modules.go` — `task test ./internal/handler/`
- [ ] 2. Реализовать `ApproveRelease` в `internal/handler/releases.go` — `task test ./internal/handler/`
- [ ] 3. Реализовать `DeleteStaticInstance` в `internal/handler/nodes.go` — `task test ./internal/handler/`
- [ ] 4. Реализовать `CreateNodeGroup` в `internal/handler/nodes.go` — `task test ./internal/handler/`

После всех subtasks: `task build` + `task lint`

---

## T-6 (CODE): G4+G5 — Composite, polling и Config хендлеры

*_Requirements: REQ-4.3–4.7, REQ-4.11–4.14, REQ-5.1–5.3_*  
*_Preservation: CP-1 (нет парциального состояния в RemoveNode), CP-4 (polling проверяет SI перед запуском), CP-5 (polling всегда завершается), CP-6 (timedOut репортится), CP-3+CP-11 (ClusterConfiguration read-only, verbatim). Поведение AddWorkerNode (P0) сохраняется после рефакторинга на shared helper._*

GOAL: Вынести `pollStaticInstance` helper из `AddWorkerNode`, реализовать `WaitNodeReady` и `RemoveNode` в NodesHandler; создать `ConfigHandler` с `GetClusterConfiguration`; зарегистрировать ConfigAPI в `main.go`.

IMPORTANT: `pollStaticInstance` — unexported метод `(h *NodesHandler)`, принимает `ctx`, `name`, `timeout`, `interval`; возвращает `(phase, elapsed, timedOut, error)` (ADR-1). `AddWorkerNode` после рефакторинга должен вызывать его.  
IMPORTANT: `RemoveNode` — последовательность: GetStaticInstance → проверка что найден → если drain: CordonNode → evict pods → DeleteStaticInstance (ADR-2). Ошибка eviction не прерывает: продолжать, считать пропущенные.  
IMPORTANT: `GetClusterConfiguration` читает Secret `d8-cluster-configuration` в namespace `kube-system`, ключ `cluster-configuration.yaml`; возвращает verbatim string (ADR-5).  
CRITICAL: После рефакторинга `AddWorkerNode` — немедленно запустить `task test ./internal/handler/` и убедиться что P0-тесты `AddWorkerNode` GREEN.

*_Test_Style:_* Tier 2 — `internal/handler/nodes_test.go`

Subtasks:
- [ ] 1. Вынести `pollStaticInstance` helper и обновить `AddWorkerNode` в `internal/handler/nodes.go` — `task test ./internal/handler/` (P0 тесты должны быть GREEN)
- [ ] 2. Реализовать `WaitNodeReady` (использует `pollStaticInstance`) в `internal/handler/nodes.go` — `task test ./internal/handler/`
- [ ] 3. Реализовать `RemoveNode` в `internal/handler/nodes.go` — `task test ./internal/handler/`
- [ ] 4. Создать `internal/handler/config.go`: `ConfigHandler` struct + `NewConfigHandler` constructor + `GetClusterConfiguration` — `task test ./internal/handler/`
- [ ] 5. Обновить `cmd/deckhouse-mcp/main.go`: создать `configHandler`, вызвать `pb.RegisterConfigAPITools(server, configHandler)` — `task build`

После всех subtasks: `task test` (весь suite) — все новые тесты GREEN, P0 тесты GREEN.

---

## T-7 (CODE): Обновление RBAC

*_Requirements: REQ-6.4_*  
*_Preservation: Существующие P0 permissions не удаляются._*

GOAL: Расширить RBAC в `deploy/rbac.yaml` для поддержки всех P1 K8s операций.

IMPORTANT: Добавлять минимально необходимые verbs. Secret `d8-cluster-configuration` — отдельный `RoleBinding` (не `ClusterRole`) в namespace `kube-system` для доступа к конкретному ресурсу.

Subtasks:
- [ ] 1. Добавить permissions для `events` (get/list/watch), `pods/log` (get), `nodes` (get/list/watch/update/patch), `secrets` (get — resourceNames: d8-cluster-configuration) в `deploy/rbac.yaml`
- [ ] 2. Добавить permissions для `moduleconfigs` (get/list/watch/create/update/patch), `deckhouserelease` (get/list/watch/update/patch), `staticinstances` (+delete), `nodegroups` (+create) в `deploy/rbac.yaml`

---

## T-8 (GATE): Checkpoint — полная верификация

*_Requirements: ALL_*

CRITICAL: Это финальное задание. Выполнять только после полного завершения T-1–T-7.

Instructions:
1. `task test` — убедиться что 100% тестов GREEN (новых 13 хендлеров + все P0)
2. `task build` — сборка без ошибок
3. `task lint` — нет нарушений proto lint rules
4. Проверить coverage matrix: каждый REQ-1.x–REQ-6.5 покрыт хотя бы одним passing тестом
5. Проверить что `config.mcp.go` существует и содержит `RegisterConfigAPITools`
6. Проверить что `deploy/rbac.yaml` содержит все P1 permissions из Design §2.9
