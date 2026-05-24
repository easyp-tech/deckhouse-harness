# Exploration: P3 — Edge Cases

## Intent

Закрыть оставшиеся 4 handler'а из roadmap (`@/Users/zergslaw/Projects/Sipki-Tech/deckhouse-mcp/ROADMAP.md`) и довести каталог Deckhouse MCP API до **43/43 handlers**. P3 — это нишевые / debugging операции, не критичные для типичных сценариев, но необходимые для полного покрытия:

- **B6 `SetModuleMaintenance`** — перевод модуля в maintenance mode (full reconciliation pause без disable)
- **D13 `CreateNodeGroupConfiguration`** — создание bash-скриптов для bootstrap нод (`NodeGroupConfiguration` CRD)
- **F3 `DeleteModuleSource`** — удаление `ModuleSource` (cleanup ненужных репозиториев модулей) с защитой от случайного удаления через `force` flag
- **F6 `ListModuleReleases`** — список доступных версий модуля (`ModuleRelease` CRD) с обязательным фильтром по `module_name`

Триггер: финализация roadmap после успешного завершения P0 (10), P1 (13) и P2 (16). Greenfield для новых CRDs (`NodeGroupConfiguration`, `ModuleRelease`), brownfield для модификаций `ModuleConfig` и `ModuleSource`.

## Investigation

Изучено (с опорой на `.spec/` docs и проектную память из работы над P2):

### Существующая инфраструктура (используется как foundation)

- **`internal/k8s/client.go`** — 8 GVR констант: `NodeGroupGVR`, `StaticInstanceGVR`, `SSHCredentialsGVR`, `ModuleConfigGVR`, `DeckhouseReleaseGVR`, `ModuleGVR`, `ModuleSourceGVR`, `ModuleUpdatePolicyGVR`. **Нужно +2**: `NodeGroupConfigurationGVR`, `ModuleReleaseGVR`.
- **`internal/k8s/client.go` методы** — около 35 методов на `Client` interface. Для P3 reusable:
  - `PatchModuleConfig` (B6 переиспользует из P2 `UpdateModuleSettings`)
  - паттерн `Delete*` (F3 будет аналогичен `DeleteSSHCredentials` из P2)
- **Proto файлы** — все 6 services готовы: `DiagnosticsAPI`, `ModulesAPI`, `ReleasesAPI`, `NodesAPI`, `ConfigAPI`, `SourcesAPI`. P3 расширяет `ModulesAPI` (B6), `NodesAPI` (D13), `SourcesAPI` (F3, F6).
- **RBAC** — `deploy/rbac.yaml` имеет least-privilege rules. P3 добавит permissions для 2 новых CRDs + delete для `modulesources`.
- **Тесты** — паттерн `Test{Handler}_{Method}_{Scenario}` + mock `k8s.Client` через function-fields в `mock_client_test.go`. 115 тестов на master.

### Deckhouse API specifics

- **`NodeGroupConfiguration` CRD** (`deckhouse.io/v1alpha1`, cluster-scoped): содержит bash-script для bootstrap. Required fields: `spec.content` (bash code), `spec.nodeGroups` (список target NodeGroups), `spec.weight` (порядок выполнения).
- **`ModuleRelease` CRD** (`deckhouse.io/v1alpha1`, cluster-scoped): перечисляет версии модулей доступные через `ModuleSource`. Labels включают `module=<name>` (FK to Module) и `source=<name>` (FK to ModuleSource) для фильтрации.
- **Maintenance mode для ModuleConfig**: реализуется через поле `spec.maintenance` в ModuleConfig spec. Семантика: **full reconciliation pause** — модуль остаётся `enabled`, но Deckhouse прекращает применять изменения (стандарт ecosystem: ArgoCD `syncPolicy=manual`, Flux `suspend: true`). `[ASSUMPTION: точное имя поля spec.maintenance и его значения подтверждаются 5-min спайком в task-plan phase (kubectl explain moduleconfig.spec в живом Kind+Deckhouse кластере)]`.

### Регрессионные ограничения

- **P0/P1/P2 не должны сломаться** — 115 unit-тестов GREEN на master, integration tools/list = 39. После P3 должно быть 43.
- **Никаких изменений в P0/P1/P2 handler'ах** — P3 ортогонален.
- **API contract совместимость** — все существующие tools остаются.

## Build Tooling

- **Orchestrator:** `go-task` (`Taskfile.yml`)
- **Test:** `task test` → `go test ./...` (115 tests, ~181s)
- **Build:** `task build` → `go build ./cmd/deckhouse-mcp`
- **Lint:** `task lint` → `easyp lint` (0 issues)
- **Generate:** `task generate` → `easyp mod download && easyp generate` (proto → `.pb.go` + `.mcp.go`)
- **Integration:** `task integration` → setup Kind+Deckhouse CE, run integration tests, teardown
- **Source:** `@/Users/zergslaw/Projects/Sipki-Tech/deckhouse-mcp/Taskfile.yml`

## Options Considered

### Option A: Single-batch implementation (recommended)

Все 4 handler'а в одной фазе разработки (одна последовательность RED→GREEN→CODE).

- **Pros:** P3 = всего 4 handler'а, не требует разбиения; единый CHANGELOG entry; один merge commit; минимальный coordination overhead
- **Cons:** Если возникает блокер на одном handler'е — задерживает остальные
- **Complexity:** Low. ~3-4 часа реализации + ~2 часа тестов (по аналогии с P2 Batch 3 — 4 handler'а SourcesAPI заняли ~6 часов wall-clock)

### Option B: Two batches by domain

Batch 1: B6 (ModuleConfig patch) + D13 (новый CRD NodeGroupConfiguration). Batch 2: F3 (DeleteModuleSource) + F6 (новый CRD ModuleRelease).

- **Pros:** Domain-grouping; легче review per batch
- **Cons:** Overhead — 2 PR, 2 CHANGELOG entries; не оправдано для 4 handler'ов
- **Complexity:** Same total time + 30 min coordination

### Option C: One-by-one (sequential)

Каждый handler — отдельная commit'ом / phase.

- **Pros:** Максимальная изоляция при review
- **Cons:** Чрезмерный overhead для edge cases; 4 раза проходить generate/lint/build
- **Complexity:** +1-2 часа из-за переключений контекста

## Design Decisions (resolved)

### B6 SetModuleMaintenance — семантика

**Решение:** реализовать как **full reconciliation pause** (вариант B из обсуждения).

- Tool description (для AI agent): *«Pauses all reconciliation; module remains enabled but Deckhouse stops applying changes — useful for debugging or manual intervention. To resume, call again with `enabled=false`.»*
- Proto: `SetModuleMaintenanceRequest { string module_name = 1; bool enabled = 2; }` — `enabled=true` входит в maintenance, `enabled=false` выходит
- Implementation: JSON Merge Patch на `ModuleConfig.spec.maintenance` (точное имя поля подтверждается спайком)
- Test cases: idempotency (enter → enter — no error), exit semantics (exit когда не в maintenance — no-op)

### F3 DeleteModuleSource — safety

**Решение:** safe-by-default с `force` flag (паттерн `kubectl --cascade=orphan`).

- Proto:
  ```proto
  message DeleteModuleSourceRequest {
    string name = 1;            // required
    optional bool force = 2;    // default false
  }
  ```
- Логика handler'а:
  - **`force=false`** (default): сначала `ListModuleReleases(source=<name>)`. Если есть active releases → ошибка `module source 'X' has N active releases (e.g., Y, Z); pass force=true to delete anyway`
  - **`force=true`**: прямой DELETE, доверяем Deckhouse owner references на cascade cleanup
- Обоснование: AI agent не удалит источник по ошибке + получит информативный error message с количеством зависимых releases

### F6 ListModuleReleases — payload и фильтры

**Решение:** compact `ModuleReleaseInfo` + REQUIRED фильтр по `module_name`.

- Proto:
  ```proto
  message ListModuleReleasesRequest {
    string module_name = 1;       // REQUIRED — фильтр по labels["module"]
    optional string phase = 2;     // optional фильтр (Pending|Deployed|Superseded|Suspended)
  }
  message ModuleReleaseInfo {
    string name = 1;       // metadata.name (e.g., "deckhouse-1.70.0")
    string module = 2;     // labels["module"]
    string version = 3;    // spec.version
    string source = 4;     // labels["source"] (FK to ModuleSource)
    string phase = 5;      // status.phase
    string approved = 6;   // spec.approved
  }
  ```
- Обоснование: компактный payload (~150-200 байт на release vs 2-5 KB полный spec) — экономия токенов LLM при больших списках; консистентность с P0/P1 паттерном `List*` (compact) vs `Get*` (detailed); REQUIRED `module_name` предотвращает unintentional full-list dumps
- Deferred (v2): `GetModuleRelease(name)` для случаев когда нужен полный spec — не входит в P3 scope

## Constraints & Risks

- **Maintenance mode field name** — точное имя поля в `ModuleConfig.spec` (предположительно `maintenance`) подтверждается **5-min спайком** в task-plan phase (`kubectl explain moduleconfig.spec` в Kind+Deckhouse CE). Запасной план: если field называется иначе (например, `suspended`), переименовываем переменные в коде — proto API не меняется (наш `enabled` параметр абстрагирует имя в Kubernetes).
- **NodeGroupConfiguration security** — handler позволяет inject bash-кода, выполняемого на нодах с root правами. RBAC для авторизованных операторов; ServiceAccount `deckhouse-mcp` уже cluster-scoped (admin-level by design, без доп. ограничений).
- **ModuleRelease scope** — REQUIRED `module_name` filter решает проблему больших списков (см. F6 решение выше).
- **Breaking changes** — отсутствуют, все P3 RPCs дополняют существующие services.
- **Dependencies** — никаких новых Go зависимостей; используется existing `dynamic.Interface` для unstructured CRDs.

## Recommended Direction

**Option A — single batch**. P3 = 4 handler'а небольшой сложности, дублирующие проверенные паттерны P0-P2. Один commit, один merge, один CHANGELOG entry «[Unreleased] — P3 — Edge Cases».

Последовательность реализации (зависимости):

1. **F6 `ListModuleReleases`** — новый CRD, нужен для F3 (используется в pre-check)
2. **F3 `DeleteModuleSource`** — зависит от F6 для force-check
3. **D13 `CreateNodeGroupConfiguration`** — независимая зона
4. **B6 `SetModuleMaintenance`** — последним (зависит от подтверждения maintenance field name через спайк)

## Scope Boundaries

- **Must-have (v1):** все 4 P3 handler'а (B6, D13, F3, F6), их proto RPCs, unit-тесты, RBAC расширение, CHANGELOG, ROADMAP.md update (отметить P3 как done)
- **Deferred (v2):**
  - `GetModuleRelease(name)` (детальный read для F блока) — для случаев когда нужен полный spec
  - `UpdateModuleSource` / `UpdateModuleUpdatePolicy` (write для F блока)
  - P0/P1 bugfix фича (7 кейсов из integration tests) — отдельная фича
  - P2 smoke-тесты в `tests/integration/test.sh` — отдельная техдолг фича
- **Needs spike:**
  - Maintenance mode field в `ModuleConfig.spec` — 5-минутный `kubectl explain moduleconfig.spec` в живом Kind+Deckhouse CE кластере (выполняется на старте task-plan phase)

## Assumptions & Open Questions

**Assumptions:**

- `[ASSUMPTION: Maintenance mode для ModuleConfig реализован через поле spec.maintenance со значениями enabled/disabled (или подобный boolean) — подтверждается 5-min спайком в начале task-plan phase. Если поле называется иначе, переименовываем только Go-переменные; proto API (parameter "enabled") абстрагирует имя]`
- `[ASSUMPTION: NodeGroupConfiguration CRD имеет structure spec.{content, nodeGroups, weight} как описано в Deckhouse docs; могут быть дополнительные поля (bundles, os) — пройдут через optional fields в proto]`
- `[ASSUMPTION: ModuleRelease CRD имеет labels "module" и "source" — стандартная Kubernetes label-based filtering применима]`
- `[ASSUMPTION: Deckhouse корректно настраивает owner references между ModuleSource → ModuleRelease, что обеспечивает cascade delete при force=true]`

**Open Questions:**

None — все 3 предыдущих open questions разрешены в Design Decisions выше.

---

## Spike result (T-6, 2026-05-16)

**Maintenance mode field name CONFIRMED via Deckhouse public docs:**

- Field path: `spec.maintenance` (string enum)
- Active value: `"NoResourceReconciliation"`
- Inactive: empty string `""` or field absent
- APIVersion: `deckhouse.io/v1alpha1`

Sources:
1. https://deckhouse.io/products/kubernetes-platform/documentation/v1/cr.html — "Defines the module maintenance mode. NoResourceReconciliation: A mode for developing or tweaking the module"
2. https://deckhouse.io/products/kubernetes-platform/documentation/v1/architecture/module-development/development/ — example: `spec: { enabled: true, maintenance: NoResourceReconciliation, settings: ... }`

JSON merge patch shape:
- Enable: `{"spec":{"maintenance":"NoResourceReconciliation"}}`
- Disable: `{"spec":{"maintenance":null}}` (RFC 7396 removes field)

`kubectl explain moduleconfig.spec` was NOT executed (Docker Desktop down; carried over from P2 session). Public docs provide equivalent authoritative answer.
