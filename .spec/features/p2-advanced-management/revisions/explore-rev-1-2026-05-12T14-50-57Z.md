# Exploration: P2 — Advanced Management

## Намерение

Реализовать 16 handler'ов фазы P2 (Advanced Management), которые разблокируют: детальную диагностику событий/логов нод, редактирование настроек модулей, операции cordon/drain/uncordon, управление версией Kubernetes, а также новый домен — управление `ModuleSource` и `ModuleUpdatePolicy`.

Триггер: завершение P0 (10/10) и P1 (13/13), переход к следующей фазе по ROADMAP.

Тип: **brownfield** — расширение существующей архитектуры новыми handlers/RPCs/k8s.Client методами без изменения ядра.

## Исследование кодовой базы

### Текущее состояние (P0 + P1)

| Метрика | Значение |
|---------|----------|
| Всего tools | 23 (5 блоков: A–E) |
| k8s.Client методов | 17 |
| Handler файлов | 5 (`diagnostics.go`, `modules.go`, `releases.go`, `nodes.go`, `config.go`) |
| Unit тестов | 70 |
| Proto сервисов | 6 (из них `SourcesAPI` — пустой stub) |

### Анализ P2 handlers по блокам

#### Блок A: Diagnostics (3 handler'а, read-only)

| ID | Handler | k8s.Client метод | Статус метода |
|----|---------|------------------|---------------|
| A4 | `GetNodeEvents` | `ListNodeEvents(ctx, nodeName)` | ✅ **уже реализован** — `client.go:280` |
| A8 | `GetStaticInstance` | `GetStaticInstance(ctx, name)` | ✅ **уже реализован** — в k8s.Client interface |
| A10 | `GetPodLogs` | `GetPodLogs(ctx, ns, pod, container, tail, since)` | ✅ **уже реализован** — `client.go:227` |

**Вывод:** Все 3 k8s.Client метода уже существуют. Нужны только proto RPC + handler method + тесты.

#### Блок B: Modules (2 handler'а)

| ID | Handler | k8s.Client метод | Статус метода |
|----|---------|------------------|---------------|
| B5 | `UpdateModuleSettings` | `UpdateModuleConfig` (reuse) | ✅ существует — update уже используется в Enable/Disable |
| B7 | `ListModules` | **новый**: `ListModules(ctx)` | ❌ нужен новый CRD `Module` (deckhouse.io) |

**Важный нюанс B7:** `Module` — это отдельный CRD от `ModuleConfig`. `Module` представляет сам модуль (runtime-состояние), а `ModuleConfig` — его конфигурацию. Это новый GVR: `deckhouse.io/v1alpha1/modules`.

**B5 (UpdateModuleSettings):** Сложность в том, что нужно мержить `spec.settings` — это произвольный объект (`map[string]any`). Подход: `GetModuleConfig` → patch settings → `UpdateModuleConfig`. Можно использовать существующий `UpdateModuleConfig`.

#### Блок D: Nodes (5 handler'ов)

| ID | Handler | k8s.Client метод | Статус метода |
|----|---------|------------------|---------------|
| D3 | `DeleteSSHCredentials` | **новый**: `DeleteSSHCredentials(ctx, name)` | ❌ нужен |
| D7 | `CordonNode` | `CordonNode(ctx, name)` | ✅ **уже реализован** — `client.go:208` |
| D8 | `UncordonNode` | **новый**: `UncordonNode(ctx, name)` | ❌ нужен (обратная операция CordonNode) |
| D9 | `DrainNode` | `CordonNode` + `ListPods` + `DeletePod` | ✅ все методы есть (composite handler) |
| D11 | `DeleteNodeGroup` | **новый**: `DeleteNodeGroup(ctx, name)` | ❌ нужен |

**D7 (CordonNode) особенность:** `CordonNode` уже есть как k8s.Client метод и используется внутри `RemoveNode`, но нет отдельного MCP tool для прямого вызова. Нужно только proto + handler.

**D8 (UncordonNode):** Обратная операция — `spec.unschedulable = false`. Новый метод в k8s.Client.

**D9 (DrainNode) — composite:** cordon → list pods on node → evict/delete non-DS pods → wait. Аналог `RemoveNode` без финального удаления StaticInstance. Все k8s.Client методы уже есть (`CordonNode`, `ListPods`, `DeletePod`).

#### Блок E: Config (2 handler'а)

| ID | Handler | k8s.Client метод | Статус метода |
|----|---------|------------------|---------------|
| E2 | `GetStaticClusterConfiguration` | `GetSecret` (reuse) | ✅ есть — другой ключ в том же секрете |
| E3 | `UpdateKubernetesVersion` | **новый**: `UpdateSecret(ctx, ns, name, secret)` | ❌ нужен |

**E2:** В секрете `d8-cluster-configuration` есть ключ `cluster-configuration.yaml` (уже читается E1) и ключ `static-cluster-configuration.yaml`.

**E3 (UpdateKubernetesVersion):** Опасная операция — модификация YAML в секрете. Нужно: прочитать секрет → распарсить YAML → изменить `kubernetesVersion` → записать обратно. Требует `update` права на секрет `d8-cluster-configuration`.

#### Блок F: Sources (4 handler'а) — НОВЫЙ ДОМЕН

| ID | Handler | k8s.Client метод | Статус метода |
|----|---------|------------------|---------------|
| F1 | `ListModuleSources` | **новый**: `ListModuleSources(ctx)` | ❌ нужен + GVR |
| F2 | `CreateModuleSource` | **новый**: `CreateModuleSource(ctx, obj)` | ❌ нужен |
| F4 | `ListModuleUpdatePolicies` | **новый**: `ListModuleUpdatePolicies(ctx)` | ❌ нужен + GVR |
| F5 | `CreateModuleUpdatePolicy` | **новый**: `CreateModuleUpdatePolicy(ctx, obj)` | ❌ нужен |

**Полностью новый домен.** Нужны:
- 2 новых GVR: `ModuleSourceGVR` (`deckhouse.io/v1alpha1/modulesources`), `ModuleUpdatePolicyGVR` (`deckhouse.io/v1alpha1/moduleupdatepolicies`)
- 4 новых метода в `k8s.Client`
- Новый `sources.go` handler файл
- Новый `sources_test.go`
- Proto RPCs в `sources.proto` (сейчас пустой stub)
- Интеграционные тесты: новые CRD-дефиниции в `tests/integration/crds.yaml`

### k8s.Client — итого новых методов

| # | Метод | Тип | Причина |
|---|-------|-----|---------|
| 1 | `UncordonNode(ctx, name) error` | typed | D8 |
| 2 | `DeleteSSHCredentials(ctx, name) error` | dynamic | D3 |
| 3 | `DeleteNodeGroup(ctx, name) error` | dynamic | D11 |
| 4 | `UpdateSecret(ctx, ns, name, *corev1.Secret) error` | typed | E3 |
| 5 | `ListModules(ctx) ([]unstructured.Unstructured, error)` | dynamic | B7 |
| 6 | `ListModuleSources(ctx) ([]unstructured.Unstructured, error)` | dynamic | F1 |
| 7 | `CreateModuleSource(ctx, obj) (*unstructured.Unstructured, error)` | dynamic | F2 |
| 8 | `ListModuleUpdatePolicies(ctx) ([]unstructured.Unstructured, error)` | dynamic | F4 |
| 9 | `CreateModuleUpdatePolicy(ctx, obj) (*unstructured.Unstructured, error)` | dynamic | F5 |

**Итого: 9 новых методов.** Клиент вырастет с 17 до 26 методов.

### RBAC — новые permissions

| Resource | APIGroup | Verbs | Reason |
|----------|----------|-------|--------|
| `pods/eviction` | `""` (core) | `create` | D9 — DrainNode eviction |
| `secrets` (d8-cluster-configuration) | `""` (core) | `update` | E3 — UpdateKubernetesVersion |
| `nodes` | `""` (core) | `patch` | D7/D8 — если CordonNode использует patch вместо update |
| `sshcredentials` | `deckhouse.io` | `delete` | D3 |
| `nodegroups` | `deckhouse.io` | `delete` | D11 |
| `modules` | `deckhouse.io` | `get`, `list` | B7 |
| `modulesources` | `deckhouse.io` | `get`, `list`, `create` | F1, F2 |
| `moduleupdatepolicies` | `deckhouse.io` | `get`, `list`, `create` | F4, F5 |

### Паттерны, которые уже установлены

1. **Handler pattern:** `struct{client k8s.Client}` + constructor → реиспользовать.
2. **Composite handlers:** `AddWorkerNode`, `RemoveNode` — `DrainNode` будет аналогичным.
3. **Idempotent writes:** `EnableModule`/`DisableModule` — `CordonNode`/`UncordonNode` аналогичны.
4. **Mock pattern:** function-field mock, 17 полей → вырастет до 26.
5. **Proto codegen:** `protoc-gen-mcp` + `easyp` → все proto определения генерятся одной командой.

### Существующие тесты, которые не должны ломаться

- 70 unit тестов в `internal/handler/` — все должны проходить.
- Integration tests (`task integration`) — Kind-кластер.
- `task lint` — easyp lint должен проходить после новых proto.
- `task generate` — должен генерировать без ошибок.

## Build Tooling

- **Оркестратор:** `task` (go-task / Taskfile.yml)
- **Тесты:** `task test` → `go test ./...`
- **Сборка:** `task build` → `go build ./cmd/deckhouse-mcp`
- **Lint:** `task lint` → `easyp lint`
- **Генерация:** `task generate` → `easyp mod download && easyp generate`
- **Docker:** `task docker:build`, `task docker:load`
- **Интеграция:** `task integration`
- **Источник:** `Taskfile.yml`

## Рассмотренные подходы

### Опция A: Все 16 handler'ов в одном цикле

Реализовать все 16 handler'ов P2 единым пакетом.

- **Плюсы:** Один release, одна фаза, полная функциональность P2 сразу.
- **Минусы:** Очень большой scope (16 RPCs + 9 k8s.Client методов + ~50+ тестов + RBAC + integration). Высокий риск merge-конфликтов, долгий review.
- **Сложность:** Высокая.

### Опция B: Разбить P2 на 3 суб-батча

1. **Batch 1 — Read-only расширения (6 handlers):** A4, A8, A10 (Diagnostics reads), B7 (ListModules), E2 (StaticClusterConfig), + D7 (CordonNode — k8s.Client уже есть).
2. **Batch 2 — Write-операции Nodes + Modules (6 handlers):** D3, D8, D9, D11 (Nodes writes), B5 (UpdateModuleSettings), E3 (UpdateKubernetesVersion).
3. **Batch 3 — Новый домен Sources (4 handlers):** F1, F2, F4, F5 — полностью новый `SourcesAPI`.

- **Плюсы:** Инкрементальная поставка, каждый batch тестируем и мержим отдельно. Batch 1 — low risk (read-only). Batch 3 изолирован (новый домен).
- **Минусы:** 3 цикла review. Немного дольше формально.
- **Сложность:** Средняя за каждый batch.

## Ограничения и риски

1. **E3 (UpdateKubernetesVersion) — высокий risk.** Изменение `d8-cluster-configuration` Secret может повлиять на работу всего кластера. Нужна валидация версии и rollback-стратегия. `[ASSUMPTION: Deckhouse валидирует kubernetesVersion при reconciliation и откатит некорректное значение]`.

2. **B7 (ListModules) — новый CRD.** `Module` (deckhouse.io/v1alpha1/modules) — отдельный от `ModuleConfig`. Нужно верифицировать GVR. `[ASSUMPTION: GVR = deckhouse.io/v1alpha1/modules]`.

3. **F-блок — полностью новый домен.** Нет предыдущего кода для `ModuleSource` / `ModuleUpdatePolicy`. Нужно исследовать CRD-схему из Deckhouse docs. `[ASSUMPTION: GVR = deckhouse.io/v1alpha1/modulesources и deckhouse.io/v1alpha1/moduleupdatepolicies]`.

4. **D9 (DrainNode) — composite + blocking.** Аналогично `RemoveNode`, но без delete. Eviction через `DeletePod` или через Eviction API (`pods/eviction`). `[ASSUMPTION: используем Eviction API для graceful drain, а не прямое удаление подов]`.

5. **Mock client explosion.** Рост с 17 до 26 function-fields в `mockClient`. Не является блокером, но увеличивает boilerplate.

6. **Обратная совместимость.** Все P0+P1 tools продолжают работать без изменений. Новые RPCs добавляются в существующие proto services.

## Рекомендованное направление

**Опция B (3 суб-батча)** — рекомендуется.

Обоснование:
- P2 содержит 16 handlers — это слишком много для одного цикла SDD (task plan + implementation + review).
- Batch 1 (read-only) — безрисковый, позволяет сразу расширить диагностику.
- Batch 3 (Sources) — изолированный новый домен, не влияет на существующий код.
- Batch 2 (writes) содержит `UpdateKubernetesVersion` — самую рискованную операцию; лучше изолировать.

## Scope Boundaries

### Must-have (v1) — Все 16 handlers P2:

**Batch 1 — Read-only (6):**
- A4: `GetNodeEvents`
- A8: `GetStaticInstance`
- A10: `GetPodLogs` (MCP tool, k8s.Client уже есть)
- B7: `ListModules`
- D7: `CordonNode` (MCP tool, k8s.Client уже есть)
- E2: `GetStaticClusterConfiguration`

**Batch 2 — Writes (6):**
- B5: `UpdateModuleSettings`
- D3: `DeleteSSHCredentials`
- D8: `UncordonNode`
- D9: `DrainNode` (composite)
- D11: `DeleteNodeGroup`
- E3: `UpdateKubernetesVersion`

**Batch 3 — Sources domain (4):**
- F1: `ListModuleSources`
- F2: `CreateModuleSource`
- F4: `ListModuleUpdatePolicies`
- F5: `CreateModuleUpdatePolicy`

### Deferred (v2 / P3):
- B6: `SetModuleMaintenance`
- D13: `CreateNodeGroupConfiguration`
- F3: `DeleteModuleSource`
- F6: `ListModuleReleases`

### Needs spike:
- **E3 (UpdateKubernetesVersion):** Требуется исследовать формат YAML в секрете `d8-cluster-configuration` и определить, как Deckhouse валидирует изменения `kubernetesVersion`. Spike можно провести в рамках Batch 2.
- **B7 (ListModules):** Верифицировать GVR для CRD `Module` (deckhouse.io).
- **F-блок:** Верифицировать CRD-схемы `ModuleSource` и `ModuleUpdatePolicy` — поля, версии, namespace scope.

## Предположения и открытые вопросы

### Предположения:
- `[ASSUMPTION: GVR для Module = deckhouse.io/v1alpha1/modules]`
- `[ASSUMPTION: GVR для ModuleSource = deckhouse.io/v1alpha1/modulesources]`
- `[ASSUMPTION: GVR для ModuleUpdatePolicy = deckhouse.io/v1alpha1/moduleupdatepolicies]`
- `[ASSUMPTION: DrainNode использует Eviction API (pods/eviction) для graceful drain]`
- `[ASSUMPTION: UpdateKubernetesVersion модифицирует YAML в Secret d8-cluster-configuration, а Deckhouse reconciler применяет изменения]`
- `[ASSUMPTION: Все 16 P2 handlers реализуются в рамках одного feature branch (worktree), но с 3 мержами по батчам]`

### Решения (согласовано):
1. **Батчинг:** ✅ 3 суб-батча (read-only → writes → Sources domain).
2. **DrainNode:** ✅ Eviction API (`pods/eviction: create`). Уважает PDB, graceful shutdown. Если PDB блокирует — возвращаем ошибку, не форсим.
3. **UpdateKubernetesVersion:** ✅ Без серверной валидации. Доверяем Deckhouse reconciler'у.
