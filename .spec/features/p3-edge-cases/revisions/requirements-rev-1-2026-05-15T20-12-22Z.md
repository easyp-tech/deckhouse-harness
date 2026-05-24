# P3 — Edge Cases — Requirements

**Status:** Draft
**Author:** Cascade AI (agent) / sipki-tech
**Date:** 2026-05-15

## Overview

P3 — Edge Cases замыкает roadmap MCP Server для Deckhouse Kubernetes Platform, добавляя 4 handler'а для нишевых административных сценариев: management `ModuleConfig` maintenance mode, создание `NodeGroupConfiguration` (bash-скрипты bootstrap нод), удаление `ModuleSource` с защитой от случайного caскад-удаления, и листинг `ModuleRelease` (доступные версии модулей). После завершения P3 каталог достигнет **43/43 handlers** (10 P0 + 13 P1 + 16 P2 + 4 P3) и `tools/list` будет возвращать все 43 tool descriptors.

## Glossary

| Term | Definition | Code Artifact |
|------|------------|---------------|
| `ModuleRelease` | Deckhouse CRD `modulereleases.deckhouse.io/v1alpha1`, представляет конкретную версию модуля доступную через `ModuleSource` | `internal/k8s/client.go` (новый GVR), `proto/deckhouse/v1/sources.proto` |
| `NodeGroupConfiguration` | Deckhouse CRD `nodegroupconfigurations.deckhouse.io/v1alpha1`, содержит bash-скрипт выполняемый на нодах целевых NodeGroups | `internal/k8s/client.go` (новый GVR), `proto/deckhouse/v1/nodes.proto` |
| Maintenance mode | Состояние `ModuleConfig`, при котором Deckhouse прекращает reconciliation модуля без его disable. Реализуется через поле `spec.maintenance` (точное имя подтверждается task-plan спайком) | `proto/deckhouse/v1/modules.proto`, `internal/handler/modules.go` |
| `force` flag | Optional boolean параметр `DeleteModuleSourceRequest`, отключающий pre-check на active `ModuleRelease`. Default — false (safe-by-default) | `proto/deckhouse/v1/sources.proto` |

## User Stories

- As a **cluster operator**, I want to list available versions of a specific module so that I can decide which one to deploy or pin via `ModuleUpdatePolicy`.
- As a **cluster operator**, I want to delete an unused `ModuleSource` safely so that I do not accidentally break modules that depend on its releases.
- As a **platform engineer**, I want to register a `NodeGroupConfiguration` script via MCP so that I can automate node bootstrap customization through AI-assisted workflows.
- As a **module maintainer**, I want to put a module into maintenance mode so that I can debug or manually intervene in its resources without Deckhouse reverting my changes.

## Requirements

### Block 1: F6 ListModuleReleases (Sources, read)

**REQ-1.1** WHEN AI agent вызывает `deckhouse_ListModuleReleases` с непустым `module_name`, the system SHALL вернуть список `ModuleRelease` ресурсов, отфильтрованных по labels `module=<module_name>`, с полями `name`, `module`, `version`, `source`, `phase`, `approved`.

**REQ-1.2** WHEN запрос содержит optional `phase` filter с непустым значением, the system SHALL дополнительно отфильтровать результат по `status.phase` равному переданному значению (case-sensitive).

**REQ-1.3** WHEN ни один `ModuleRelease` не соответствует фильтрам, the system SHALL вернуть пустой список без ошибки.

**REQ-1.4** WHEN `module_name` пустой или отсутствует, the system SHALL вернуть ошибку валидации `module_name is required`, не делая запросов к Kubernetes API.

### Block 2: F3 DeleteModuleSource (Sources, write)

**REQ-2.1** WHEN AI agent вызывает `deckhouse_DeleteModuleSource` с `force=false` (или unset) и существуют `ModuleRelease` ресурсы с label `source=<name>`, the system SHALL вернуть ошибку с сообщением вида `module source 'X' has N active releases (e.g., Y, Z); pass force=true to delete anyway`, **не удаляя** `ModuleSource`.

**REQ-2.2** WHEN AI agent вызывает `deckhouse_DeleteModuleSource` с `force=false` и нет ни одного `ModuleRelease` с label `source=<name>`, the system SHALL удалить `ModuleSource` и вернуть `success=true`.

**REQ-2.3** WHEN AI agent вызывает `deckhouse_DeleteModuleSource` с `force=true`, the system SHALL удалить `ModuleSource` напрямую без pre-check, доверяя Deckhouse owner references на cascade cleanup `ModuleRelease`.

**REQ-2.4** WHEN указанный `ModuleSource` не существует, the system SHALL вернуть ошибку `not found` независимо от значения `force`.

### Block 3: D13 CreateNodeGroupConfiguration (Nodes, write)

**REQ-3.1** WHEN AI agent вызывает `deckhouse_CreateNodeGroupConfiguration` с `name`, `content` (bash-script), `node_groups` (список имён) и optional `weight`, the system SHALL создать `NodeGroupConfiguration` ресурс в кластере с указанными полями.

**REQ-3.2** WHEN `weight` не передан, the system SHALL установить значение по умолчанию `100` (середина допустимого диапазона 1-200) для совместимости с Deckhouse-реализацией.

**REQ-3.3** WHEN `NodeGroupConfiguration` с таким `name` уже существует, the system SHALL вернуть ошибку `already exists`, не модифицируя существующий ресурс.

**REQ-3.4** WHEN `content` пустой или `node_groups` пустой список, the system SHALL вернуть ошибку валидации до запроса к Kubernetes API.

### Block 4: B6 SetModuleMaintenance (Modules, write)

**REQ-4.1** WHEN AI agent вызывает `deckhouse_SetModuleMaintenance` с `module_name` и `enabled=true`, the system SHALL установить поле `spec.maintenance` (или эквивалент по итогам спайка) в `ModuleConfig` для перевода модуля в режим full reconciliation pause, и вернуть `previousState=enabled|disabled`.

**REQ-4.2** WHEN AI agent вызывает `deckhouse_SetModuleMaintenance` с `module_name` и `enabled=false`, the system SHALL снять поле `spec.maintenance` (или установить его в значение «disabled» согласно Deckhouse API) для возобновления reconciliation модуля.

**REQ-4.3** WHEN модуль уже находится в запрошенном состоянии (вызов `enabled=true` в maintenance, или `enabled=false` вне maintenance), the system SHALL вернуть `success=true` и `previousState`, выполнив идемпотентный no-op patch (или явный no-op без вызова API).

**REQ-4.4** WHEN указанный `ModuleConfig` не существует, the system SHALL вернуть ошибку `not found`.

### Block 5: Non-functional Requirements

**REQ-5.1** WHEN добавляется любой новый handler, the system SHALL расширить `k8s.Client` интерфейс (`internal/k8s/client.go`) новыми методами **до** реализации handler-логики (process invariant из P0/P1/P2).

**REQ-5.2** WHEN добавляется новый k8s.Client метод, the system SHALL добавить соответствующее function-field в `mockClient` в `internal/handler/mock_client_test.go`.

**REQ-5.3** WHEN добавляется handler, требующий новые K8s permissions, the system SHALL обновить `deploy/rbac.yaml` минимально необходимыми правами (least-privilege; no wildcards; resourceName-scoping где применимо).

**REQ-5.4** WHEN добавляется handler, работающий с новым CRD, the system SHALL добавить GVR-константу в `internal/k8s/client.go`. Для P3 это `NodeGroupConfigurationGVR` и `ModuleReleaseGVR`.

**REQ-5.5** WHEN все 4 handler'а реализованы, the system SHALL проходить полный test suite `go test ./...` без ошибок (текущая база — 115 тестов; ожидаемое количество после P3 — приблизительно 143).

**REQ-5.6** WHEN все handler'ы реализованы, the system SHALL проходить `task generate` и `task lint` без ошибок.

**REQ-5.7** WHEN feature merged в master, the system SHALL обновить `ROADMAP.md` отметив P3 как `✅ Done (4/4)` в Implementation Order таблице и Phase progress tracker.

**REQ-5.8** WHEN feature merged в master, the system SHALL добавить секцию «[Unreleased] — P3 — Edge Cases» в `CHANGELOG.md` с перечислением всех 4 новых tools и инфраструктурных изменений.

## Verification Commands

Источник: exploration document §Build Tooling. Команды используются verbatim во всех последующих фазах.

| Action | Command | Expected Result |
|--------|---------|-----------------|
| Generate | `task generate` | exit 0; `easyp generate` reports no errors |
| Lint | `task lint` | exit 0; `easyp lint` reports `issues=0` |
| Test | `task test` | exit 0; all packages PASS (~143 tests after P3, ~181-200s) |
| Build | `task build` | exit 0; binary `deckhouse-mcp` создан |
| Integration | `task integration` | exit 0; tools/list = 43 (10 P0 + 13 P1 + 16 P2 + 4 P3) |

## Open Design Questions

None — все разрешены в `.spec/features/p3-edge-cases/explore.md` § «Design Decisions (resolved)». Единственный остающийся пункт — 5-минутный спайк по точному имени поля maintenance mode в `ModuleConfig.spec` — выполняется в начале task-plan phase, не требует решения на уровне requirements.

## Conflicts and Resolutions

None identified. Все 4 handler'а ортогональны (разные блоки, разные CRDs) и не конфликтуют между собой или с P0/P1/P2 функциональностью.
