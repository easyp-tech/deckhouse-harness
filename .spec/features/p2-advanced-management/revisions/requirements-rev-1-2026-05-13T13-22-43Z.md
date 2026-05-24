# P2 — Advanced Management: Requirements

**Статус:** Draft
**Дата:** 2026-05-12

## Обзор

Расширение Deckhouse MCP Server 16 handler'ами фазы P2 (Advanced Management). Добавляется детальная диагностика (события нод, логи подов), управление настройками модулей, операции cordon/drain/uncordon, управление версией Kubernetes, а также новый домен — `ModuleSource` и `ModuleUpdatePolicy`. Реализация в 3 последовательных батчах. Существующие P0+P1 handler'ы, proto-определения и тесты **не модифицируются**.

## Глоссарий

| Термин | Определение | Code Artifact |
|--------|-------------|---------------|
| `Eviction` | Graceful удаление пода через Eviction API, уважающее PodDisruptionBudget | `internal/handler/nodes.go` |
| `ModuleSource` | CRD Deckhouse, определяющий OCI-реестр с модулями | `proto/deckhouse/v1/sources.proto` |
| `ModuleUpdatePolicy` | CRD Deckhouse, определяющий политику автообновления модулей | `proto/deckhouse/v1/sources.proto` |
| `DrainTimeout` | Максимальное время ожидания drain-операции (300s) | `internal/handler/nodes.go` |

## User Stories

- As an **AI agent**, I want to read node events and pod logs so that I can diagnose cluster problems without switching to kubectl.
- As an **AI agent**, I want to cordon, drain, and uncordon nodes so that I can perform node maintenance through MCP tools.
- As an **AI agent**, I want to enable/disable modules and update their settings so that I can manage Deckhouse configuration conversationally.
- As an **AI agent**, I want to manage ModuleSources and ModuleUpdatePolicies so that I can configure module delivery pipelines.
- As an **AI agent**, I want to update the Kubernetes version so that I can orchestrate cluster upgrades.

---

## Требования

### Группа 1 — Diagnostics (read-only расширения)

**REQ-1.1** WHEN AI agent вызывает `deckhouse_GetNodeEvents` с именем ноды, the system SHALL вернуть список событий Kubernetes для этой ноды, отсортированных по времени.

**REQ-1.2** WHEN указанная нода не существует, the system SHALL вернуть ошибку `not found`.

**REQ-1.3** WHEN AI agent вызывает `deckhouse_GetStaticInstance` с именем StaticInstance, the system SHALL вернуть полный spec и status ресурса, включая адрес, фазу, credentialsRef и связанную ноду.

**REQ-1.4** WHEN указанный StaticInstance не существует, the system SHALL вернуть ошибку `not found`.

**REQ-1.5** WHEN AI agent вызывает `deckhouse_GetPodLogs` с namespace, именем пода и опциональными параметрами (container, tail, since), the system SHALL вернуть текст логов указанного пода.

**REQ-1.6** WHEN под не существует или namespace не найден, the system SHALL вернуть ошибку с описанием причины.

### Группа 2 — Modules (расширение)

**REQ-2.1** WHEN AI agent вызывает `deckhouse_ListModules`, the system SHALL вернуть список всех `Module` ресурсов (CRD `deckhouse.io`) с именем, весом, источником и состоянием.

**REQ-2.2** WHEN AI agent вызывает `deckhouse_UpdateModuleSettings` с именем модуля и map настроек, the system SHALL обновить `spec.settings` в соответствующем ModuleConfig, сохранив все поля, не указанные в запросе.

**REQ-2.3** WHEN указанный ModuleConfig не существует, the system SHALL вернуть ошибку `not found`.

**REQ-2.4** WHEN переданные настройки пусты (пустой map), the system SHALL вернуть ошибку валидации, не модифицируя ресурс.

### Группа 3 — Nodes (cordon/drain/uncordon)

**REQ-3.1** WHEN AI agent вызывает `deckhouse_CordonNode` с именем ноды, the system SHALL установить `spec.unschedulable = true` на ноде и вернуть предыдущее состояние. Операция идемпотентна.

**REQ-3.2** WHEN AI agent вызывает `deckhouse_UncordonNode` с именем ноды, the system SHALL установить `spec.unschedulable = false` на ноде и вернуть предыдущее состояние. Операция идемпотентна.

**REQ-3.3** WHEN указанная нода не существует, the system SHALL вернуть ошибку `not found` для CordonNode и UncordonNode.

**REQ-3.4** WHEN AI agent вызывает `deckhouse_DrainNode` с именем ноды, the system SHALL выполнить cordon ноды, затем evict все поды (кроме DaemonSet и mirror pods) через Eviction API.

**REQ-3.5** WHEN PodDisruptionBudget блокирует eviction пода, the system SHALL повторять попытки в пределах таймаута (по умолчанию 300 секунд).

**REQ-3.6** WHEN таймаут drain истёк, the system SHALL вернуть частичный результат с `timedOut: true` и списком подов, которые не удалось evict.

**REQ-3.7** WHEN AI agent вызывает `deckhouse_DeleteSSHCredentials` с именем ресурса, the system SHALL удалить SSHCredentials. Операция необратима.

**REQ-3.8** WHEN SSHCredentials не существует, the system SHALL вернуть ошибку `not found`.

**REQ-3.9** WHEN AI agent вызывает `deckhouse_DeleteNodeGroup` с именем NodeGroup, the system SHALL удалить ресурс.

**REQ-3.10** WHEN NodeGroup не существует, the system SHALL вернуть ошибку `not found`.

### Группа 4 — Config (конфигурация кластера)

**REQ-4.1** WHEN AI agent вызывает `deckhouse_GetStaticClusterConfiguration`, the system SHALL прочитать ключ `static-cluster-configuration.yaml` из Secret `d8-cluster-configuration` в namespace `kube-system` и вернуть содержимое как строку.

**REQ-4.2** WHEN ключ `static-cluster-configuration.yaml` отсутствует в секрете, the system SHALL вернуть ошибку с описанием.

**REQ-4.3** WHEN AI agent вызывает `deckhouse_UpdateKubernetesVersion` с целевой версией, the system SHALL прочитать Secret `d8-cluster-configuration`, распарсить YAML `cluster-configuration.yaml`, изменить поле `kubernetesVersion` и записать обновлённый секрет обратно. Валидация версии не производится — доверяется Deckhouse reconciler.

**REQ-4.4** WHEN секрет `d8-cluster-configuration` не найден, the system SHALL вернуть ошибку.

### Группа 5 — Sources (новый домен)

**REQ-5.1** WHEN AI agent вызывает `deckhouse_ListModuleSources`, the system SHALL вернуть список всех `ModuleSource` ресурсов с именем, URI реестра и статусом.

**REQ-5.2** WHEN AI agent вызывает `deckhouse_CreateModuleSource` с именем и URI реестра, the system SHALL создать ресурс ModuleSource.

**REQ-5.3** WHEN ModuleSource с таким именем уже существует, the system SHALL вернуть ошибку `already exists`.

**REQ-5.4** WHEN AI agent вызывает `deckhouse_ListModuleUpdatePolicies`, the system SHALL вернуть список всех `ModuleUpdatePolicy` ресурсов.

**REQ-5.5** WHEN AI agent вызывает `deckhouse_CreateModuleUpdatePolicy` с именем и параметрами политики, the system SHALL создать ресурс ModuleUpdatePolicy.

**REQ-5.6** WHEN ModuleUpdatePolicy с таким именем уже существует, the system SHALL вернуть ошибку `already exists`.

### Группа 6 — Инфраструктура (сквозные требования)

**REQ-6.1** WHEN добавляется любой новый handler, the system SHALL расширить `k8s.Client` интерфейс новыми методами до реализации handler'а.

**REQ-6.2** WHEN добавляется новый k8s.Client метод, the system SHALL добавить соответствующее function-field в `mockClient` в `mock_client_test.go`.

**REQ-6.3** WHEN добавляется handler, требующий новые K8s permissions, the system SHALL обновить `deploy/rbac.yaml` с минимально необходимыми правами.

**REQ-6.4** WHEN добавляется новый CRD (ModuleSource, ModuleUpdatePolicy, Module), the system SHALL добавить GVR-константу в `internal/k8s/client.go`.

**REQ-6.5** WHEN все handler'ы батча реализованы, the system SHALL проходить все существующие тесты (`task test`) без ошибок.

**REQ-6.6** WHEN все handler'ы батча реализованы, the system SHALL проходить `task generate` и `task lint` без ошибок.

---

## Топологический порядок

```
Batch 1 (read-only): REQ-1.* → REQ-2.1
  Reason: Read-only handlers не имеют зависимостей, можно реализовывать параллельно.

Batch 2 (writes): REQ-2.2–2.4, REQ-3.*, REQ-4.*
  Зависимость: REQ-3.4 (DrainNode) зависит от REQ-3.1 (CordonNode).
  REQ-4.3 зависит от REQ-4.1 (GetStaticClusterConfiguration использует тот же секрет).

Batch 3 (Sources): REQ-5.*
  Reason: Полностью изолированный домен. Зависит от REQ-6.4 (GVR-константы).

Сквозные (REQ-6.*): применяются к каждому батчу.
```

Порядок батчей строго последовательный: Batch 1 → Batch 2 → Batch 3.

---

## Приоритет конфликтов

```
REQ-3.4 (DrainNode выполняет cordon) vs REQ-3.1 (CordonNode как отдельный tool).
Резолюция: DrainNode вызывает cordon как первый шаг. CordonNode существует как standalone tool
для случаев, когда drain не нужен. Не конфликт — разная гранулярность.
```

---

## Verification Commands

| Действие | Команда | Источник |
|----------|---------|----------|
| Тесты | `task test` | Taskfile.yml |
| Сборка | `task build` | Taskfile.yml |
| Lint | `task lint` | Taskfile.yml |
| Генерация | `task generate` | Taskfile.yml |
| Docker | `task docker:build` | Taskfile.yml |
| Интеграция | `task integration` | Taskfile.yml |
