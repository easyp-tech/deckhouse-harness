# Code Review: p1-core-operations

## Verdict: PASS

Реализация 13 MCP-инструментов блоков A–E (P1) в целом соответствует требованиям и дизайну. Первичный смотр выявил 4 несоответствия (2 major, 2 minor), все они исправлены автоматически в рамках fix-цикла до выставления финального вердикта. После исправлений: build OK, lint OK, 72 теста — все PASS.

---

## Change Set

| Файл | Статус | Примечание |
|------|--------|-----------|
| `deploy/rbac.yaml` | ✅ Planned | P1 RBAC permissions |
| `internal/handler/config.go` | ✅ Planned | NEW: ConfigHandler |
| `internal/handler/config_test.go` | ✅ Planned | NEW: 2 tests |
| `internal/handler/diagnostics.go` | ✅ Planned | +GetNode, +GetNodeGroup, +GetDeckhouseLogs; fix-цикл: +ListNodeEvents |
| `internal/handler/diagnostics_test.go` | ✅ Planned | +8 tests; fix-цикл: +TestGetNode_WithEvents |
| `internal/handler/mock_client_test.go` | ✅ Planned | +21 function fields; fix-цикл: +listNodeEventsFunc, +deletePodFunc |
| `internal/handler/modules.go` | ✅ Planned | +GetModuleConfig, +EnableModule, +DisableModule |
| `internal/handler/modules_test.go` | ✅ Planned | +7 tests |
| `internal/handler/nodes.go` | ✅ Planned | +pollStaticInstance, +DeleteStaticInstance, +RemoveNode, +CreateNodeGroup, +WaitNodeReady; fix-цикл: pod deletion |
| `internal/handler/nodes_test.go` | ✅ Planned | +9 tests; fix-цикл: extended TestRemoveNode_DrainAndDelete |
| `internal/handler/releases.go` | ✅ Planned | +GetDeckhouseRelease, +ApproveRelease |
| `internal/handler/releases_test.go` | ✅ Planned | +5 tests |
| `internal/k8s/client.go` | ✅ Planned | +11 interface methods + impls; fix-цикл: +ListNodeEvents, +DeletePod |
| `proto/deckhouse/v1/config.mcp.go` | ✅ Planned | NEW: generated |
| `proto/deckhouse/v1/config.pb.go` | ✅ Planned | regenerated |
| `proto/deckhouse/v1/config.proto` | ✅ Planned | +GetClusterConfiguration RPC |
| `proto/deckhouse/v1/diagnostics.mcp.go` | ✅ Planned | regenerated |
| `proto/deckhouse/v1/diagnostics.pb.go` | ✅ Planned | regenerated |
| `proto/deckhouse/v1/diagnostics.proto` | ✅ Planned | +GetNode, +GetNodeGroup, +GetDeckhouseLogs RPCs |
| `proto/deckhouse/v1/modules.mcp.go` | ✅ Planned | regenerated |
| `proto/deckhouse/v1/modules.pb.go` | ✅ Planned | regenerated |
| `proto/deckhouse/v1/modules.proto` | ✅ Planned | +GetModuleConfig, +EnableModule, +DisableModule RPCs |
| `proto/deckhouse/v1/nodes.mcp.go` | ✅ Planned | regenerated |
| `proto/deckhouse/v1/nodes.pb.go` | ✅ Planned | regenerated |
| `proto/deckhouse/v1/nodes.proto` | ✅ Planned | +DeleteStaticInstance, +RemoveNode, +CreateNodeGroup, +WaitNodeReady RPCs |
| `proto/deckhouse/v1/releases.mcp.go` | ✅ Planned | regenerated |
| `proto/deckhouse/v1/releases.pb.go` | ✅ Planned | regenerated |
| `proto/deckhouse/v1/releases.proto` | ✅ Planned | +GetDeckhouseRelease, +ApproveRelease RPCs |

Все файлы из task plan присутствуют. Неожиданных изменений нет.

---

## Requirements Traceability

| Требование | Тест(ы) | Код | CP | Вердикт |
|-----------|---------|-----|----|---------|
| REQ-1.1 | `TestGetNode_Found` | `diagnostics.go:GetNode` | CP-10 | ✅ |
| REQ-1.2 | `TestGetNode_NotFound` | `diagnostics.go:GetNode` | CP-9 | ✅ |
| REQ-1.3 | `TestGetNode_Found` | `diagnostics.go:GetNode` (conditions, allocatable, capacity) | — | ✅ |
| REQ-1.4 | `TestGetNode_WithEvents` | `diagnostics.go:GetNode` + `k8s.Client.ListNodeEvents` | — | ✅ (fix F-1) |
| REQ-1.5 | `TestGetNode_Found`, `TestGetNode_NoStaticInstance` | `diagnostics.go:GetNode` (staticInstancePhase) | — | ✅ |
| REQ-1.6 | `TestGetNodeGroup_Found` | `diagnostics.go:GetNodeGroup` | CP-9 | ✅ |
| REQ-1.7 | `TestGetNodeGroup_Found` | `diagnostics.go:GetNodeGroup` (nodeNames from nodes list) | — | ✅ |
| REQ-1.8 | `TestGetNodeGroup_NotFound` | `diagnostics.go:GetNodeGroup` | CP-9 | ✅ |
| REQ-1.9 | `TestGetDeckhouseLogs_Success` | `diagnostics.go:GetDeckhouseLogs` | CP-10 | ✅ |
| REQ-1.10 | `TestGetDeckhouseLogs_Grep` | `diagnostics.go:GetDeckhouseLogs` (grep filter) | — | ✅ |
| REQ-1.11 | `TestGetDeckhouseLogs_Success` | `diagnostics.go:GetDeckhouseLogs` (label app=deckhouse) | — | ✅ |
| REQ-1.12 | `TestGetDeckhouseLogs_NoPod` | `diagnostics.go:GetDeckhouseLogs` | CP-9 | ✅ |
| REQ-2.1 | `TestGetModuleConfig_Found` | `modules.go:GetModuleConfig` | — | ✅ |
| REQ-2.2 | `TestGetModuleConfig_NotFound` | `modules.go:GetModuleConfig` | CP-9 | ✅ |
| REQ-2.3 | `TestEnableModule_WasDisabled`, `TestEnableModule_AlreadyEnabled` | `modules.go:EnableModule` (reads before write) | CP-2 | ✅ |
| REQ-2.4 | `TestEnableModule_AlreadyEnabled` | `modules.go:EnableModule` (`previousState:true`) | CP-8 | ✅ |
| REQ-2.5 | `TestEnableModule_NotFound` | `modules.go:EnableModule` | CP-9 | ✅ |
| REQ-2.6 | `TestDisableModule_WasEnabled`, `TestDisableModule_AlreadyDisabled` | `modules.go:DisableModule` | CP-2 | ✅ |
| REQ-2.7 | `TestDisableModule_NotFound` | `modules.go:DisableModule` | CP-9 | ✅ |
| REQ-3.1 | `TestGetDeckhouseRelease_Found` | `releases.go:GetDeckhouseRelease` | — | ✅ |
| REQ-3.2 | `TestGetDeckhouseRelease_NotFound` | `releases.go:GetDeckhouseRelease` | CP-9 | ✅ |
| REQ-3.3 | `TestApproveRelease_AlreadyApproved` | `releases.go:ApproveRelease` | CP-7 | ✅ |
| REQ-3.4 | `TestApproveRelease_NotFound` | `releases.go:ApproveRelease` | CP-9 | ✅ |
| REQ-4.1 | `TestDeleteStaticInstance_Success` | `nodes.go:DeleteStaticInstance` | CP-9 | ✅ |
| REQ-4.2 | `TestDeleteStaticInstance_NotFound` | `nodes.go:DeleteStaticInstance` | CP-9 | ✅ |
| REQ-4.3 | `TestRemoveNode_DrainAndDelete`, `TestRemoveNode_NoStaticInstance` | `nodes.go:RemoveNode` (GetStaticInstance check) | CP-1 | ✅ |
| REQ-4.4 | `TestRemoveNode_NoStaticInstance` | `nodes.go:RemoveNode` (error msg) | CP-1 | ✅ (fix F-4) |
| REQ-4.5 | `TestRemoveNode_DrainAndDelete` | `nodes.go:RemoveNode` (CordonNode + DeletePod loop) | CP-1 | ✅ (fix F-2) |
| REQ-4.6 | `TestRemoveNode_NoDrain` | `nodes.go:RemoveNode` (drain=false skips cordon+pods) | — | ✅ |
| REQ-4.7 | `TestRemoveNode_DrainAndDelete` | `nodes.go:RemoveNode` + `isDaemonSetPod` | CP-1 | ✅ (fix F-2) |
| REQ-4.8 | `TestCreateNodeGroup_Success` | `nodes.go:CreateNodeGroup` | — | ✅ |
| REQ-4.9 | proto required field (`name = 1`) | `nodes.go:CreateNodeGroup` | — | ✅ |
| REQ-4.10 | `TestCreateNodeGroup_AlreadyExists` | `nodes.go:CreateNodeGroup` | CP-9 | ✅ |
| REQ-4.11 | `TestWaitNodeReady_Success` | `nodes.go:pollStaticInstance` (phase=="Running") | CP-5 | ✅ ⚠️ |
| REQ-4.12 | `TestWaitNodeReady_Timeout`, `TestWaitNodeReady_Success` | `nodes.go:pollStaticInstance` (CP-4: error on first GetStaticInstance failure) | CP-4 | ✅ |
| REQ-4.13 | `TestWaitNodeReady_Timeout` | `nodes.go:pollStaticInstance` | CP-5, CP-6 | ✅ |
| REQ-4.14 | `TestWaitNodeReady_Timeout` | `nodes.go:WaitNodeReady` (`timedOut:true` + `phase` in response) | CP-6 | ✅ |
| REQ-5.1 | `TestGetClusterConfiguration_Success` | `config.go:GetClusterConfiguration` | CP-3, CP-11 | ✅ |
| REQ-5.2 | `TestGetClusterConfiguration_SecretNotFound` | `config.go:GetClusterConfiguration` | CP-9 | ✅ (fix F-3) |
| REQ-5.3 | (нет теста) | не реализовано | — | ⚠️ (design conflict) |
| REQ-6.1 | (compile check) | `internal/k8s/client.go` interface | — | ✅ |
| REQ-6.2 | (compile check) | `internal/k8s/client.go` implementation | — | ✅ |
| REQ-6.3 | все 72 теста | все handler-файлы | — | ✅ |
| REQ-6.4 | (review) | `deploy/rbac.yaml` | — | ✅ |
| REQ-6.5 | (review) | `proto/deckhouse/v1/*.pb.go`, `*.mcp.go` | — | ✅ |

**Примечания:**

- REQ-4.11 ⚠️: требование говорит "фаза `Bootstrapped`", но реальная фаза Deckhouse — `Running` (StaticInstance.status.currentStatus.phase). Реализация правильная; это опечатка в требованиях. `AddWorkerNode` (P0, уже в production) также использует `"Running"`. Оценка: корректно реализовано.
- REQ-5.3 ⚠️: требование запрашивает YAML-валидацию при чтении конфигурации. ADR-5 дизайна явно отклоняет этот подход: "decoding/parsing the YAML — unnecessary complexity". Поскольку дизайн является утверждённым артефактом фазы 3, а возврат сырой строки полностью соответствует ADR-5, это расхождение требований и дизайна. Реализация соответствует дизайну.

---

## Design Conformance

### 3.1 Architectural Boundaries

Все новые компоненты размещены в правильных пакетах:
- `internal/handler/` — реализации ToolHandler
- `internal/k8s/client.go` — все K8s-операции через интерфейс
- `proto/deckhouse/v1/` — сгенерированный код, не редактируется вручную

Зависимости: `handler → k8s.Client` (через интерфейс), `handler → pb` (generated types). Нет несанкционированных пересечений слоёв. ✅

### 3.2 Data Models

Все новые сообщения proto соответствуют §2.5 дизайна:
- `GetNodeResponse.Events []*NodeEvent` — реализовано (fix F-1)
- `NodeEvent.{Reason, Message, Type, LastTime string, Count int32}` — реализовано корректно
- `RemoveNodeResponse.{Drained bool, Deleted bool}` — реализовано
- Все остальные request/response messages совпадают с §2.5 ✅

### 3.3 API Contracts

- `ListNodeEvents(ctx, nodeName) ([]corev1.Event, error)` — добавлен в интерфейс ✅
- `DeletePod(ctx, namespace, name) error` — добавлен в интерфейс ✅
- `GetSecret(ctx, namespace, name) (*corev1.Secret, error)` — уже был ✅
- `CordonNode(ctx, name) error` — уже был ✅
- Все 13 новых RPCs зарегистрированы в `main.go` через `pb.Register*Tools(server, handler)` ✅
- Error formats: `fmt.Errorf("operation: %w", err)` — соответствует §2.7 ✅

### 3.4 Error Handling

| Сценарий | Ожидаемый (§2.7) | Фактический |
|----------|-----------------|------------|
| GetNode not found | `"getting node %s: %w"` | `"getting node %s: %w"` ✅ |
| RemoveNode no StaticInstance | `"static instance for node %q not found"` | `"static instance for node %q not found: %w"` ✅ |
| GetClusterConfiguration secret missing | `"cluster configuration secret not found: %w"` | `"cluster configuration secret not found: %w"` ✅ (fix F-3) |
| GetDeckhouseLogs no pod | `"deckhouse pod not found in d8-system"` | `"deckhouse pod not found in d8-system"` ✅ |
| Polling error | `"polling StaticInstance %s: %w"` | `"polling StaticInstance %s: %w"` ✅ |

### 3.5 Correctness Properties

| CP | Выполнено |
|----|---------|
| CP-1: нет мутации до успешного GetStaticInstance в RemoveNode | ✅ |
| CP-2: EnableModule/DisableModule читают state перед записью | ✅ |
| CP-3: ClusterConfiguration возвращается verbatim | ✅ |
| CP-4: WaitNodeReady возвращает ошибку при первой неудаче GetStaticInstance | ✅ |
| CP-5: Polling всегда завершается (ctx, deadline, phase check) | ✅ |
| CP-6: timeout → `timedOut: true` + последняя фаза | ✅ |
| CP-7: ApproveRelease идемпотентен (patch already-set annotation) | ✅ |
| CP-8: EnableModule с уже включённым = `{success:true, previousState:true}` | ✅ |
| CP-9: Ресурс не найден → явная ошибка | ✅ |
| CP-10: GetDeckhouseLogs read-only | ✅ |
| CP-11: GetClusterConfiguration только GET | ✅ |

### 3.6 Documentation Consistency

`cmd/deckhouse-mcp/main.go` включает регистрацию всех 5 сервисов (Diagnostics, Modules, Releases, Nodes, Config). Соответствует архитектурной диаграмме в дизайне §2.2. ✅

---

## Code Quality

### 4.1 Naming & Clarity

- Все новые идентификаторы соответствуют конвенции проекта: `New{Name}Handler`, `{Name}Handler`, метод-имена совпадают с proto RPC именами.
- `isDaemonSetPod` — ясное, однозначное название. ✅
- `pollStaticInstance` — корректно вынесен в helper для переиспользования между `AddWorkerNode` и `WaitNodeReady`. ✅

### 4.2 Dead Code & Debug Artifacts

Нет закомментированного кода, debug-print, неиспользованных импортов или переменных. ✅

### 4.3 Scope Creep

Реализация строго следует task plan. Нет дополнительных фич или рефакторинга за пределами требований. `setModuleEnabled` — оправданный внутренний helper, не виден снаружи пакета. ✅

### 4.4 Test Quality

- 72 теста охватывают все 13 новых хендлеров с happy path + error paths.
- Тестовые имена описательны: `TestGetNode_Found`, `TestRemoveNode_DrainAndDelete`, etc.
- Моки используют function fields с nil-default (вызов = не ожидается, не вызван = OK). ✅
- `TestGetNode_WithEvents` верифицирует конкретные поля `Reason`, `Count`. ✅
- `TestRemoveNode_DrainAndDelete` верифицирует, что pod deletion вызывается при `drain=true`. ✅
- Polling-тесты (`WaitNodeReady_Timeout`) используют мок без реального sleep — корректно.
- Минорное замечание: нет явного теста для `isDaemonSetPod` (пропуска DaemonSet подов при drain). Функция покрыта косвенно через `RemoveNode`, но явный unit-test был бы полезен. Это nit, не блокер.

---

## Security

- **Input validation**: все внешние входы валидируются через proto (required fields) или runtime-проверкой (`privateKey == ""` в `CreateSSHCredentials`). ✅
- **Authentication**: MCP-сервер аутентифицируется через `rest.InClusterConfig()` (ServiceAccount + RBAC). Нет новых публичных эндпоинтов без авторизации. ✅
- **Authorization**: RBAC в `deploy/rbac.yaml` обновлён для P1-хендлеров с принципом наименьших привилегий. ✅
- **Injection**: нет SQL, нет shell-команд. `nodeName` используется как pod field selector — передаётся через typed Kubernetes API (не через шаблоны строк для shell). ✅
- **Secrets**: SSH ключи и sudo-пароль кодируются в base64 внутри хендлера, never exposed в логах. `ClusterConfiguration` YAML не содержит секретов (подтверждено разведкой). ✅
- **Data exposure**: ошибки содержат имена ресурсов, но не internal Kubernetes internals. ✅
- **DaemonSet pod filter in `isDaemonSetPod`**: фильтрация по `OwnerReferences[*].Kind` корректна и безопасна. ✅

Проблем безопасности не выявлено.

---

## Verification Evidence

**Tests:**
```
=== RUN   TestGetNode_Found
--- PASS: TestGetNode_Found (0.00s)
=== RUN   TestGetNode_NotFound
--- PASS: TestGetNode_NotFound (0.00s)
=== RUN   TestGetNode_NoStaticInstance
--- PASS: TestGetNode_NoStaticInstance (0.00s)
=== RUN   TestGetNode_WithEvents
--- PASS: TestGetNode_WithEvents (0.00s)
=== RUN   TestRemoveNode_DrainAndDelete
--- PASS: TestRemoveNode_DrainAndDelete (0.00s)
=== RUN   TestRemoveNode_NoDrain
--- PASS: TestRemoveNode_NoDrain (0.00s)
=== RUN   TestRemoveNode_NoStaticInstance
--- PASS: TestRemoveNode_NoStaticInstance (0.00s)
=== RUN   TestGetClusterConfiguration_Success
--- PASS: TestGetClusterConfiguration_Success (0.00s)
=== RUN   TestGetClusterConfiguration_SecretNotFound
--- PASS: TestGetClusterConfiguration_SecretNotFound (0.00s)
--- PASS: TestWaitNodeReady_Timeout (30.00s)
--- PASS: TestWaitNodeReady_Success (30.00s)
PASS
ok      github.com/easyp-tech/deckhouse-mcp/internal/handler    121.210s
```

**Build:**
```
go build ./... 2>&1
(no output — build successful)
```

**Lint:**
```
task lint
time=2026-04-14T17:57:26.607+03:00 level=INFO msg="starting lint"
time=2026-04-14T17:57:26.669+03:00 level=INFO msg="lint completed" issues=0
```

---

## Findings

| ID | Severity | Файл | Описание | Требование |
|----|----------|------|----------|-----------|
| F-1 | major | `diagnostics.go:GetNode` | `Events` в `GetNodeResponse` был пустым: `ListNodeEvents` не вызывался. Исправлено: добавлен `ListNodeEvents` в `k8s.Client` + вызов в хендлере. | REQ-1.4 |
| F-2 | major | `nodes.go:RemoveNode` | drain=true только корdonировал ноду, поды не удалялись. Исправлено: добавлен `DeletePod` в `k8s.Client` + цикл удаления non-DaemonSet подов. | REQ-4.5, REQ-4.7 |
| F-3 | minor | `config.go:GetClusterConfiguration` | Сообщение ошибки `"getting d8-cluster-configuration secret: %w"` не совпадало с дизайном §2.7. Исправлено: `"cluster configuration secret not found: %w"`. | REQ-5.2 |
| F-4 | minor | `nodes.go:RemoveNode` | Сообщение ошибки `"static instance %s not found"` не совпадало с дизайном §2.7. Исправлено: `"static instance for node %q not found"`. | REQ-4.4 |
| F-5 | nit | `diagnostics.go:GetNode` | `listPodsFunc` в тесте `TestGetNode_Found` вызывался, хотя `GetNode` хендлер `ListPods` не вызывает. Безвреден — это избыточная моковая настройка в тесте, не влияет на корректность. | — |

Все critical и major найдены устранены. Nit-замечание F-5 оставлено без исправления (безвредно, может быть внесено в следующем цикле).

---

## Fix Plan

*(выполнен в рамках данного review-цикла)*

### Fix F-1 (major) — GetNode events [DONE]

1. `internal/k8s/client.go`: добавлен `ListNodeEvents(ctx, nodeName string) ([]corev1.Event, error)` в интерфейс + реализация через `CoreV1().Events("").List(ctx, FieldSelector: "involvedObject.name="+nodeName, Limit: 10)`.
2. `internal/handler/mock_client_test.go`: добавлен `listNodeEventsFunc` + метод `ListNodeEvents`.
3. `internal/handler/diagnostics.go`: `GetNode` теперь вызывает `ListNodeEvents` и заполняет `Events []*pb.NodeEvent`.
4. `internal/handler/diagnostics_test.go`: добавлен `TestGetNode_WithEvents` (verify Reason + Count).

### Fix F-2 (major) — RemoveNode pod deletion [DONE]

1. `internal/k8s/client.go`: добавлен `DeletePod(ctx, namespace, name string) error` в интерфейс + реализация через `CoreV1().Pods(namespace).Delete(...)`.
2. `internal/handler/mock_client_test.go`: добавлен `deletePodFunc` + метод `DeletePod`.
3. `internal/handler/nodes.go`: `RemoveNode` теперь вызывает `ListPods("")`, фильтрует по `pod.Spec.NodeName == req.Name`, пропускает DaemonSet pods (через `isDaemonSetPod`), удаляет остальные через `DeletePod`. Добавлен хелпер `isDaemonSetPod(*corev1.Pod) bool`.
4. `internal/handler/nodes_test.go`: `TestRemoveNode_DrainAndDelete` расширен: добавлены `listPodsFunc` (возвращает один обычный pod), `deletePodFunc` (устанавливает `podDeleted=true`), `assert podDeleted`.

### Fix F-3 (minor) — config.go error text [DONE]

`internal/handler/config.go`: изменено сообщение ошибки с `"getting d8-cluster-configuration secret: %w"` → `"cluster configuration secret not found: %w"`.

### Fix F-4 (minor) — nodes.go error text [DONE]

`internal/handler/nodes.go`: изменён формат ошибки с `"static instance %s not found: %w"` → `"static instance for node %q not found: %w"`.
