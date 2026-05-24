# Exploration: P1 Core Operations

## Намерение

Реализовать 13 MCP-хендлеров уровня P1 (Core Operations), которые охватывают ~80% реальных сценариев управления кластером: детальный просмотр нод/групп/модулей/релизов, включение/выключение модулей, подтверждение обновления Deckhouse, управление жизненным циклом static-нод и NodeGroup, получение логов Deckhouse-контроллера, ожидание готовности ноды, и чтение ClusterConfiguration.

Все 13 хендлеров ждёт 0/13 в `ROADMAP.md` — это следующий логический шаг после MVP (P0).

## Исследование

### Существующая архитектура

**Задокументирована в `.spec/`** — основные выводы ниже.

Паттерн P0 полностью унифицирован:
1. `.proto` файл → `task generate` → `*.pb.go` + `*.mcp.go`
2. Реализовать метод в `internal/handler/{service}.go`
3. Добавить методы к `k8s.Client` в `internal/k8s/client.go`
4. Написать тесты в `internal/handler/{service}_test.go` с `mockClient`
5. Обновить `deploy/rbac.yaml`

### Анализ 13 хендлеров P1

| ID | Handler | Сложность | Особенности |
|----|---------|-----------|-------------|
| A3 | `GetNode` | средняя | Node detail + 10 последних Events + связанный SI phase |
| A6 | `GetNodeGroup` | низкая | Один CRD unstructured + список нод группы |
| A11 | `GetDeckhouseLogs` | средняя | `pods/log` API — deckhouse pod в d8-system |
| B2 | `GetModuleConfig` | низкая | Один CRD unstructured из существующего GVR |
| B3 | `EnableModule` | средняя | Update `.spec.enabled = true` + get current state |
| B4 | `DisableModule` | средняя | Update `.spec.enabled = false` + get current state |
| C2 | `GetDeckhouseRelease` | низкая | Один CRD из существующего GVR |
| C3 | `ApproveRelease` | средняя | Patch annotation `release.deckhouse.io/approved: "true"` |
| D5 | `DeleteStaticInstance` | низкая | `dynamic.Delete()` по GVR |
| D6 | `RemoveNode` | высокая | Composite: необязательный drain + delete SI (зависит от D5) |
| D10 | `CreateNodeGroup` | средняя | Новый CRD с параметрами (nodeType, taints, labels, disruptions) |
| D12 | `WaitNodeReady` | средняя | Polling: аналог polling в `AddWorkerNode`, переиспользует логику |
| E1 | `GetClusterConfiguration` | средняя | Читает K8s Secret `d8-cluster-configuration` в `kube-system` |

### Новые методы k8s.Client

11 новых методов необходимо добавить в `internal/k8s/client.go`:

```go
GetNode(ctx, name) → (*corev1.Node, error)
ListEvents(ctx, namespace, fieldSelector) → ([]corev1.Event, error)
GetPodLogs(ctx, namespace, podName, container string, tail int64, since string) → (string, error)
GetNodeGroup(ctx, name) → (*unstructured.Unstructured, error)
GetModuleConfig(ctx, name) → (*unstructured.Unstructured, error)
UpdateModuleConfig(ctx, obj) → (*unstructured.Unstructured, error)
GetDeckhouseRelease(ctx, name) → (*unstructured.Unstructured, error)
PatchDeckhouseRelease(ctx, name, patch) → (*unstructured.Unstructured, error)
DeleteStaticInstance(ctx, name) → error
CreateNodeGroup(ctx, obj) → (*unstructured.Unstructured, error)
GetSecret(ctx, namespace, name) → (*corev1.Secret, error)
```

### Специальные случаи

**D6 RemoveNode (composite):** drain + delete StaticInstance. Drain в Kubernetes — это eviction всех pod'ов: нужен `pods/eviction` RBAC verb + `nodes` patch для cordon. По спецификации `drain` опционален (default: true). Это сложнее чем другие P1 хендлеры.

**A11 GetDeckhouseLogs:** нужно найти под с label `app=deckhouse` в namespace `d8-system`, получить логи через `pods/log` API. Поддерживает `tail`, `since`, `grep` (grep делается на стороне сервера).

**E1 GetClusterConfiguration:** ClusterConfiguration хранится в Secret `d8-cluster-configuration` в namespace `kube-system` (или `d8-system`). Нужен новый RBAC: `secrets: get`. Нужно декодировать base64 из `.data` секрета.

**D12 WaitNodeReady:** дублирует polling-логику из `AddWorkerNode.go`. Есть смысл вынести polling в общую helper-функцию внутри пакета handler.

### Новые GVR

Нет — все необходимые GVR уже определены в `internal/k8s/client.go`. `config.proto` использует K8s Secret (core resource), а не CRD.

### RBAC расширения

```yaml
# Новые разрешения для P1
- events: get, list
- pods/log: get (уже есть в rbac.yaml!)
- secrets: get (для E1)
- moduleconfigs: update, patch (B3, B4)
- deckhouserelease: update, patch (C3)
- staticinstances: delete (D5, D6)
- nodegroups: create (D10)
# D6 drain потребует:
- pods/eviction: create
- nodes: patch (cordon)
```

**Примечание:** `pods/log` и `events` уже есть в `deploy/rbac.yaml` (видимо добавлены на будущее).

### Build Tooling

- **Orchestrator:** `go-task` (`Taskfile.yml`)
- **Generate:** `task generate` (`easyp mod download && easyp generate`)
- **Build:** `task build`
- **Lint:** `task lint`
- **Test:** `task test` (`go test ./...`, ~60s)
- **Source:** `Taskfile.yml`, `easyp.yaml`

## Рассмотренные варианты

### Вариант A: Реализовать все 13 хендлеров в одном PR строго по P0-паттерну

Следовать точно той же последовательности, что и при P0: proto → generate → handler → k8s.Client → tests → rbac.

- **Плюсы:** Единообразие, минимальный риск, простое code review
- **Минусы:** Большой PR (~13 хендлеров + 11 k8s методов + 5 proto файлов), сложно ревьюить за один раз
- **Сложность:** Высокая (объём), низкая (техническая)

### Вариант B: Разбить на несколько независимых PR по блокам

A-блок, B-блок, C-блок, D-блок, E-блок — отдельно.

- **Плюсы:** Маленькие ревью, быстрый мерж, можно параллелить
- **Минусы:** `k8s.Client` interface меняется в каждом PR (merge conflicts), proto-файлы трогаются несколько раз
- **Сложность:** Средняя (управление зависимостями между PR)

### Вариант C: Реализовать только простые хендлеры (read-only), отложить write

A3, A6, B2, C2, E1 — только чтение. D5, D6, B3, B4, C3, D10, D12 — отложить.

- **Плюсы:** Минимальный риск (нет деструктивных операций), быстро
- **Минусы:** Не даёт реальной ценности — без write-операций нет смысла в Core Operations, D12 блокирует D6
- **Сложность:** Низкая, но неполная

## Ограничения и риски

- **D6 RemoveNode drain:** полный drain (eviction) — сложная операция. Deckhouse сам рекомендует cordon → drain → delete. Необходимо обработать PodDisruptionBudget и статические тома. `[ASSUMPTION: для MVP drain просто evict все poды без учёта PDB]`
- **E1 ClusterConfiguration secret name:** предполагаем `d8-cluster-configuration` в `kube-system`. `[ASSUMPTION: имя секрета и namespace совпадают с документацией Deckhouse]`
- **config.proto:** блок E пока stub без RPCs — нужно добавить первый real RPC (GetClusterConfiguration). Это не breaking change.
- **Polling refactor:** D12 и `AddWorkerNode` делают одно и то же — polling StaticInstance. `[ASSUMPTION: выносим polling в unexported helper внутри пакета handler]`
- **A11 grep:** фильтрация по подстроке на стороне сервера (strings.Contains) — не regex. `[ASSUMPTION: простого Contains достаточно для P1]`

## Рекомендуемое направление

**Вариант A** — реализовать все 13 хендлеров в рамках одной фичи, но разбить на логические группы задач в Task Plan:
1. Группа "read-only simple" (A3, A6, B2, C2) — без новых RBAC write
2. Группа "write-simple" (B3, B4, C3, D5, D10) — update/patch/delete/create
3. Группа "logs & config" (A11, E1) — special K8s ops
4. Группа "composite & polling" (D6, D12) — зависит от D5

Это даёт полный P1 за один цикл, при этом Task Plan структурирован по сложности и зависимостям.

## Границы области

- **Must-have (v1):** Все 13 хендлеров P1 из ROADMAP.md: A3, A6, A11, B2, B3, B4, C2, C3, D5, D6, D10, D12, E1
- **Deferred (v2):** P2/P3 хендлеры (DrainNode как standalone, DeleteNodeGroup, UpdateModuleSettings, etc.)
- **Needs spike:** Точное местоположение ClusterConfiguration secret (nspace + name) — нужно проверить в реальном кластере или документации. D6 drain-поведение при StatefulSet pods.

## Допущения и открытые вопросы

`[ASSUMPTION: drain в D6 MVP = cordon node + evict all pods (без учёта PDB, StatefulSet-ов)]`  
`[ASSUMPTION: ClusterConfiguration хранится в secret "d8-cluster-configuration" namespace "kube-system"]`  
`[ASSUMPTION: A11 grep — простой strings.Contains, не regexp]`  
`[ASSUMPTION: polling helper выносится в unexported func внутри пакета handler, переиспользуется в AddWorkerNode и D12]`  
`[ASSUMPTION: D10 CreateNodeGroup для CE поддерживает только nodeType: Static]`

**Открытые вопросы:**
1. D6 `RemoveNode`: если у ноды нет StaticInstance (например, cloud-нода), что делать? Только drain? Или ошибка?
2. E1: есть ли в ClusterConfiguration секретные поля (пароли, ключи), которые нужно маскировать перед отдачей?
3. D12 `WaitNodeReady`: ждём StaticInstance phase или K8s Node Ready condition? По спеке — StaticInstance phase, но для cloud-нод StaticInstance нет.
