# P1 Core Operations — Requirements

**Status:** Draft  
**Date:** 2026-04-14

## Обзор

Реализовать 13 MCP-инструментов уровня P1 (Core Operations) для сервера `deckhouse-mcp`. Инструменты покрывают детальный просмотр ресурсов кластера, управление модулями Deckhouse, подтверждение релизов, удаление static-нод и NodeGroup, получение логов Deckhouse, ожидание готовности ноды, и чтение конфигурации кластера. Реализация следует существующему proto-first паттерну (P0): `.proto` → `task generate` → handler → `k8s.Client` → тесты → RBAC. Код разбивается на атомарные коммиты — по одному хендлеру или логически связанной группе.

## Глоссарий

| Термин | Определение | Code Artifact |
|--------|-------------|---------------|
| StaticInstance | CRD Deckhouse, представляющий bare-metal/VM машину, подключённую к кластеру как K8s-нода | `internal/k8s/client.go` — `StaticInstanceGVR` |
| NodeGroup | CRD Deckhouse, определяющий группу однородных нод | `internal/k8s/client.go` — `NodeGroupGVR` |
| ModuleConfig | CRD Deckhouse, управляющий конфигурацией и состоянием модуля | `internal/k8s/client.go` — `ModuleConfigGVR` |
| DeckhouseRelease | CRD Deckhouse, представляющий доступный релиз для обновления | `internal/k8s/client.go` — `DeckhouseReleaseGVR` |
| ClusterConfiguration | Конфигурация кластера, хранящаяся в K8s Secret `d8-cluster-configuration` в namespace `kube-system` | `internal/k8s/client.go` — `GetSecret` |
| polling | Периодический опрос K8s ресурса с фиксированным интервалом до достижения целевого состояния или таймаута | `internal/handler/nodes.go` — логика `AddWorkerNode` |
| proto-first | Архитектурный паттерн: `.proto` файл — единственный источник истины для MCP-инструментов | `proto/deckhouse/v1/` |

## Пользовательские истории

- Как **оператор кластера**, я хочу получить детальную информацию по конкретной ноде, NodeGroup, ModuleConfig или DeckhouseRelease, чтобы не листать весь список для диагностики.
- Как **оператор кластера**, я хочу включать и выключать модули Deckhouse через AI-агента, чтобы управлять функциональностью без ручного редактирования YAML.
- Как **оператор кластера**, я хочу подтверждать обновление Deckhouse через AI-агента, чтобы контролировать момент применения релиза.
- Как **оператор кластера**, я хочу удалять static-ноды и NodeGroup через AI-агента, чтобы управлять инфраструктурой полного цикла.
- Как **оператор кластера**, я хочу просматривать логи Deckhouse-контроллера и ждать готовности ноды, не переключаясь на `kubectl`.

## Требования

### Блок A — Diagnostics (read-only)

**REQ-1.1** WHEN вызван инструмент `deckhouse_GetNode` с параметром `name`, the system SHALL вернуть полный статус K8s Node: все conditions, allocatable/capacity ресурсы, internal IP, kubelet version, роли.

**REQ-1.2** WHEN вызван `deckhouse_GetNode` и нода с указанным `name` не найдена, the system SHALL вернуть ошибку с текстом, содержащим имя ноды и указанием "not found".

**REQ-1.3** WHEN вызван `deckhouse_GetNode` и для ноды существует StaticInstance с тем же именем, the system SHALL включить в ответ поле `staticInstancePhase` с текущей фазой StaticInstance.

**REQ-1.4** WHEN вызван `deckhouse_GetNode`, the system SHALL включить в ответ последние 10 K8s Events, связанных с данной нодой (fieldSelector `involvedObject.name=<name>`).

**REQ-1.5** WHEN вызван инструмент `deckhouse_GetNodeGroup` с параметром `name`, the system SHALL вернуть полный spec и status NodeGroup.

**REQ-1.6** WHEN вызван `deckhouse_GetNodeGroup` и NodeGroup с указанным `name` не найдена, the system SHALL вернуть ошибку с текстом, содержащим имя группы и указанием "not found".

**REQ-1.7** WHEN вызван `deckhouse_GetNodeGroup`, the system SHALL включить в ответ список имён нод, принадлежащих данной NodeGroup (по label `node.deckhouse.io/group=<name>`).

**REQ-1.8** WHEN вызван инструмент `deckhouse_GetDeckhouseLogs` без параметров, the system SHALL вернуть последние 100 строк логов пода с label `app=deckhouse` в namespace `d8-system`.

**REQ-1.9** WHEN вызван `deckhouse_GetDeckhouseLogs` с параметром `tail`, the system SHALL вернуть не более указанного количества строк.

**REQ-1.10** WHEN вызван `deckhouse_GetDeckhouseLogs` с параметром `since` (например `"30m"`), the system SHALL вернуть только строки, записанные за указанный период.

**REQ-1.11** WHEN вызван `deckhouse_GetDeckhouseLogs` с параметром `grep`, the system SHALL вернуть только строки, содержащие указанную подстроку (case-sensitive).

**REQ-1.12** WHEN вызван `deckhouse_GetDeckhouseLogs` и под Deckhouse не найден в `d8-system`, the system SHALL вернуть ошибку "deckhouse pod not found in d8-system".

### Блок B — Modules

**REQ-2.1** WHEN вызван инструмент `deckhouse_GetModuleConfig` с параметром `name`, the system SHALL вернуть полный spec и status ModuleConfig, включая `enabled`, `version`, `settings` и `status.conditions`.

**REQ-2.2** WHEN вызван `deckhouse_GetModuleConfig` и ModuleConfig с указанным `name` не найдена, the system SHALL вернуть ошибку с текстом, содержащим имя и указанием "not found".

**REQ-2.3** WHEN вызван инструмент `deckhouse_EnableModule` с параметром `name`, the system SHALL установить поле `.spec.enabled = true` в ModuleConfig и вернуть предыдущее значение `enabled` в поле `previousState`.

**REQ-2.4** WHEN вызван `deckhouse_EnableModule` и модуль с указанным `name` уже включён (`spec.enabled == true`), the system SHALL выполнить операцию (update) и вернуть `previousState: true`.

**REQ-2.5** WHEN вызван `deckhouse_EnableModule` и ModuleConfig с указанным `name` не найдена, the system SHALL вернуть ошибку с текстом, содержащим имя и указанием "not found".

**REQ-2.6** WHEN вызван инструмент `deckhouse_DisableModule` с параметром `name`, the system SHALL установить поле `.spec.enabled = false` в ModuleConfig и вернуть предыдущее значение `enabled` в поле `previousState`.

**REQ-2.7** WHEN вызван `deckhouse_DisableModule` и ModuleConfig с указанным `name` не найдена, the system SHALL вернуть ошибку с текстом, содержащим имя и указанием "not found".

### Блок C — Releases

**REQ-3.1** WHEN вызван инструмент `deckhouse_GetDeckhouseRelease` с параметром `version` (например `"v1.74.0"`), the system SHALL вернуть полный spec и status DeckhouseRelease, включая `requirements`, `changelog` и `approved`.

**REQ-3.2** WHEN вызван `deckhouse_GetDeckhouseRelease` и релиз с указанным `version` не найден, the system SHALL вернуть ошибку с текстом, содержащим версию и указанием "not found".

**REQ-3.3** WHEN вызван инструмент `deckhouse_ApproveRelease` с параметром `version`, the system SHALL установить аннотацию `release.deckhouse.io/approved: "true"` на объекте DeckhouseRelease и вернуть предыдущее значение аннотации в поле `previousApproved`.

**REQ-3.4** WHEN вызван `deckhouse_ApproveRelease` и релиз с указанным `version` не найден, the system SHALL вернуть ошибку с текстом, содержащим версию и указанием "not found".

### Блок D — Nodes (write)

**REQ-4.1** WHEN вызван инструмент `deckhouse_DeleteStaticInstance` с параметром `name`, the system SHALL удалить объект StaticInstance с указанным именем через K8s API и вернуть подтверждение успеха.

**REQ-4.2** WHEN вызван `deckhouse_DeleteStaticInstance` и StaticInstance с указанным `name` не найден, the system SHALL вернуть ошибку с текстом, содержащим имя и указанием "not found".

**REQ-4.3** WHEN вызван инструмент `deckhouse_RemoveNode` с параметром `name`, the system SHALL найти StaticInstance с именем `name` и удалить его.

**REQ-4.4** WHEN вызван `deckhouse_RemoveNode` и StaticInstance с указанным `name` не существует, the system SHALL вернуть ошибку "static instance for node %q not found" без выполнения каких-либо удалений.

**REQ-4.5** WHEN вызван `deckhouse_RemoveNode` с параметром `drain: true` (по умолчанию), the system SHALL перед удалением StaticInstance выполнить cordon K8s Node и eviction всех pod'ов с этой ноды.

**REQ-4.6** WHEN вызван `deckhouse_RemoveNode` с параметром `drain: false`, the system SHALL удалить StaticInstance без cordon и eviction.

**REQ-4.7** WHEN при drain в `deckhouse_RemoveNode` eviction pod'а завершается ошибкой, the system SHALL продолжить eviction остальных pod'ов и вернуть в ответе количество успешно эвакуированных и пропущенных pod'ов.

**REQ-4.8** WHEN вызван инструмент `deckhouse_CreateNodeGroup` с параметрами `name` и `nodeType`, the system SHALL создать объект NodeGroup с указанными параметрами и вернуть созданный объект.

**REQ-4.9** WHEN вызван `deckhouse_CreateNodeGroup` без обязательного параметра `name`, the system SHALL вернуть ошибку валидации.

**REQ-4.10** WHEN вызван `deckhouse_CreateNodeGroup` и NodeGroup с таким `name` уже существует, the system SHALL вернуть ошибку, содержащую указание "already exists".

**REQ-4.11** WHEN вызван инструмент `deckhouse_WaitNodeReady` с параметром `name`, the system SHALL опрашивать StaticInstance каждые 30 секунд (или `intervalSeconds`) до тех пор, пока фаза не станет `"Bootstrapped"` или не истечёт таймаут.

**REQ-4.12** WHEN в `deckhouse_WaitNodeReady` StaticInstance с указанным `name` не найден, the system SHALL вернуть ошибку "static instance %q not found".

**REQ-4.13** WHEN в `deckhouse_WaitNodeReady` таймаут истекает раньше достижения фазы `"Bootstrapped"`, the system SHALL вернуть последнюю наблюдаемую фазу и поле `timedOut: true` без возврата ошибки.

**REQ-4.14** WHEN параметр `timeoutSeconds` не указан в `deckhouse_WaitNodeReady`, the system SHALL использовать значение по умолчанию 900 секунд.

### Блок E — Configuration

**REQ-5.1** WHEN вызван инструмент `deckhouse_GetClusterConfiguration`, the system SHALL прочитать K8s Secret с именем `d8-cluster-configuration` в namespace `kube-system`, декодировать YAML из поля `.data["cluster-configuration.yaml"]` и вернуть содержимое как структурированный объект.

**REQ-5.2** WHEN Secret `d8-cluster-configuration` не найден, the system SHALL вернуть ошибку "cluster configuration secret not found".

**REQ-5.3** WHEN содержимое Secret не является корректным YAML, the system SHALL вернуть ошибку с описанием проблемы парсинга.

### Общие требования

**REQ-6.1** WHEN для любого нового хендлера добавляются K8s-операции, the system SHALL выполнять все K8s-вызовы исключительно через интерфейс `k8s.Client` (не напрямую через `client-go`).

**REQ-6.2** WHEN добавляется новый метод в `k8s.Client`, the system SHALL добавить соответствующее function-field в `mockClient` и реализовать метод по существующему nil-check паттерну.

**REQ-6.3** WHEN реализован новый хендлер, the system SHALL содержать unit-тесты, покрывающие happy path и как минимум один error case.

**REQ-6.4** WHEN реализованы новые K8s-операции, требующие дополнительных RBAC-полномочий, the system SHALL обновить `deploy/rbac.yaml` с минимально необходимыми verb'ами.

**REQ-6.5** WHEN добавляются новые RPC в `.proto` файлы, the system SHALL запустить `task generate` для обновления `*.pb.go` и `*.mcp.go` перед реализацией.

## Топологический порядок

```
REQ-1.x (A3 GetNode, A6 GetNodeGroup)       — независимы
REQ-1.8–1.12 (A11 GetDeckhouseLogs)         — независим
REQ-2.1–2.2 (B2 GetModuleConfig)            — независим
REQ-2.3–2.5 (B3 EnableModule)               — зависит от GetModuleConfig (нужен Get для чтения previousState)
REQ-2.6–2.7 (B4 DisableModule)              — зависит от GetModuleConfig
REQ-3.1–3.2 (C2 GetDeckhouseRelease)        — независим
REQ-3.3–3.4 (C3 ApproveRelease)             — зависит от C2 (нужен Get для чтения previousApproved)
REQ-4.1–4.2 (D5 DeleteStaticInstance)       — независим
REQ-4.3–4.7 (D6 RemoveNode)                 — зависит от D5 (вызывает DeleteStaticInstance)
REQ-4.8–4.10 (D10 CreateNodeGroup)          — независим
REQ-4.11–4.14 (D12 WaitNodeReady)           — независим
REQ-5.1–5.3 (E1 GetClusterConfiguration)    — независим
REQ-6.x (общие)                             — применимы ко всем
```

## Открытые вопросы для фазы Design

| Вопрос | Важность | Затронутые REQ |
|--------|----------|----------------|
| Как именно извлечь список нод для NodeGroup (label query vs field selector)? | Средняя | REQ-1.7 |
| Нужно ли маскировать какие-либо поля ClusterConfiguration перед отдачей? | Низкая | REQ-5.1 |
| Какой именно ключ внутри Secret содержит ClusterConfiguration YAML? | Высокая | REQ-5.1 |

## Команды верификации

| Действие | Команда | Источник |
|----------|---------|----------|
| Тест | `task test` | `Taskfile.yml` |
| Сборка | `task build` | `Taskfile.yml` |
| Линт (proto) | `task lint` | `Taskfile.yml` |
| Генерация | `task generate` | `Taskfile.yml` |
