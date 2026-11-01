# Архитектура CARVION

Текущий код отделён здесь от целевой архитектуры. Порядок этапов указан в [ROADMAP.md](ROADMAP.md).

## Текущая архитектура — реализовано

```text
cmd/carvion-api (API/Hub transport)
  ├── config.Load + config.LoadHTTP
  ├── composition.NewLocal
  │     ├── application.Service (catalog + localization + simulator workflows)
  │     ├── infrastructure/postgres.CatalogRepository ─── PostgreSQL
  │     ├── infrastructure/postgres.LocalizationRepository ─ PostgreSQL
  │     └── infrastructure/adapterbridge ── Plugin Manager
  └── internal/server (chi handlers) ── application.Service

application.Service
  ├── domain types and errors
  ├── CatalogRepository port ── local PostgreSQL implementation
  └── AdapterGateway port ── SDK/Plugin Manager bridge
                                  ├── JSON-RPC process → adapter-android-demo
                                  └── JSON-RPC process → adapter-geely-research

desktop/main.go (Wails 3)
  ├── embedded React bundle
  └── ApplicationService bindings → application.Service directly
```

`internal/core/domain` содержит transport-neutral типы и ошибки; `internal/core/application` содержит use cases, в том числе simulator и Geely research orchestration, и порты; `internal/core/infrastructure` реализует PostgreSQL repositories и переводит контракты SDK/Manager в Core domain. Localization parsers, validator, exporters и package builder расположены в `internal/localization`; общий каталог синтетических профилей и Android XML fixture — в `internal/simulator`; versioned Geely profile/evidence/exact matcher/report — в `internal/geelyresearch`. Эти пакеты не зависят от HTTP или Wails. `internal/composition.NewLocal` создаёт pool, repositories, локальный adapter manager и application service и владеет закрытием PostgreSQL pool. API запускает этот Core в процессе, HTTP handlers вызывают его напрямую.

Wails service facade в `desktop/` вызывает `application.Service` напрямую. `cmd/carvion-api` остаётся отдельной точкой входа; Desktop не вызывает локальный HTTP API.

## Инвентаризация репозитория

| Путь | Назначение / наблюдение |
|---|---|
| `cmd/` | Точки входа root module: API, Android demo adapter и Geely research adapter; Wails app находится в `desktop/` module |
| `desktop/` | Wails Desktop entry point, Go binding facade, lifecycle и отдельный Wails Go module |
| `desktop/frontend/` | React/TypeScript интерфейс, Wails-generated bindings, UI primitives и Playwright E2E |
| `pkg/adaptersdk/v1/` | Публичный Go contract и JSON-RPC server |
| `internal/geelyresearch/` | Versioned profile schema, публичные evidence, пустой exact compatibility catalog и очищенный report |
| `internal/adapters/geelyresearch/` | Исследовательский adapter process; shared XML/JSON localization parser |
| `internal/core/domain/` | Domain types и ошибки |
| `internal/core/application/` | Application service, simulator orchestration и порты `AdapterGateway` / `CatalogRepository` |
| `internal/core/infrastructure/` | PostgreSQL catalog repository и adapter bridge к SDK/Manager |
| `internal/composition/` | Создание local runtime и lifecycle PostgreSQL pool |
| `internal/simulator/` | Embedded synthetic profile catalog и Android XML fixture, общие для Core и demo adapter |
| `internal/` | Также config, DB pool, HTTP transport, Plugin Manager, demo adapter и тесты |
| `adapters/` | Manifests Android demo и Geely research; `bin/` содержит generated executables после build |
| `db/migrations/` | Четыре Goose migrations: catalog, Adapter SDK, Localization Engine и Deployment audit |
| `docs/` | Существующие заметки SDK/архитектуры и новая проектная документация |
| `.github/workflows/ci.yml` | Go format check, vet, unit tests и build |
| `.idea/` | Локальные IDE project settings; не часть runtime architecture |
| `go.mod`, `go.sum` | Go module `github.com/Sneypsleep90/carvion` и зависимости |
| `Makefile`, `compose.yaml`, `.env.example`, `.gitignore` | Локальная сборка, PostgreSQL, пример окружения и ignored artifacts |
| `bin/` | Generated API executable после `make build`, игнорируемый Git |

В проверенном workspace доступен Go backend; наличие или состояние соседних компонентов предполагаемого monorepo `carvion` не подтверждено.

## HTTP-зависимости

| Компонент | Зависимость |
|---|---|
| `internal/server` | HTTP-specific: chi router, wire DTO/validation, status codes, logging middleware и handlers. Зависит от Core интерфейса и domain types; SDK/Manager не импортирует |
| `cmd/carvion-api` | HTTP listener, address, timeouts и graceful shutdown; Core dependencies создаёт `internal/composition` |
| `internal/config` | `Load` возвращает core/local settings; `LoadHTTP` отдельно возвращает host/port. Compose-only `POSTGRES_*` credentials в config не загружаются |
| `internal/core/domain` | Не зависит от HTTP, SDK, pgx или Wails |
| `internal/core/application` | Use cases и repository/adapter ports; не зависит от transport или инфраструктуры |
| `internal/core/infrastructure/adapterbridge` | Infrastructure boundary к Adapter SDK/Plugin Manager |
| `internal/core/infrastructure/postgres` | Infrastructure boundary к pgx/PostgreSQL |
| `pkg/adaptersdk/v1`, `internal/pluginmanager`, `internal/adapters/androiddemo` | Не зависят от HTTP и сохраняют прежние роли |
| `internal/composition` | Собирает local PostgreSQL runtime и adapters; используется API и Wails Desktop |

HTTP API остаётся отдельным transport и текущей точкой входа для Hub/API процесса. Существующие маршруты сохранены. По HTTP доступны список адаптеров и проверка совместимости; `Detect` и `InspectResources` доступны через Core application service и Desktop bindings, отдельные HTTP routes для них пока отсутствуют.

`CatalogRepository` читает `manufacturers`, `vehicle_models`, `infotainment_platforms`. `LocalizationRepository` хранит проекты, source snapshots, ресурсы, переводы, историю, validation runs и package artifacts. Adapter/firmware/compatibility tables из SDK migration сохранены, но CRUD и синхронизация с локальным manifest catalog не реализованы. Будущий remote/Hub-backed repository должен реализовать application ports; такая реализация пока отсутствует.

## Текущая Desktop-first архитектура — реализовано

```text
React + TypeScript UI
        │ generated Wails bindings (typed service facade)
        ▼
Wails 3 app in desktop/
        │ direct Go call
        ▼
ApplicationService (lifecycle + binding boundary)
        │
internal/composition.NewLocal → application.Service
  ├── domain / use cases / errors
  ├── CatalogRepository → local PostgreSQL (development)
  ├── LocalizationRepository → local PostgreSQL (development)
  ├── internal/localization → XML/JSON/CSV/validation/ZIP
  ├── internal/simulator → whitelisted synthetic profiles/resources
  ├── Geely research use cases → adapterbridge → Plugin Manager → Geely JSON-RPC subprocess
  ├── AdapterGateway → Plugin Manager → demo JSON-RPC subprocess
  └── DeploymentService → package verifier → PostgreSQL jobs/events → synthetic-only provider

cmd/carvion-api → HTTP handlers → same Core (separate Hub/API transport)
```

Wails bindings и HTTP handlers вызывают application services; Core не вызывается через локальный REST endpoint. Для разработки composition использует PostgreSQL. Application зависит от repository ports, поэтому Hub-backed implementation можно подключить без изменения use cases и без DB credentials в UI. Remote implementation и авторизация Hub ещё не реализованы. Plugin Manager остаётся границей Core и adapter processes.

Geely Research Mode использует специальный Core use case с фиксированным `adapter-geely-research` ID и дополнительным `vehicle_profile` SDK request. Профиль и результаты не сохраняются как отдельные записи; resource file попадает в Localization Engine только по действию пользователя. Report builder выводит hash/key metadata без local path и raw resource contents. Для этого процесса не добавлялись HTTP routes или DB migrations.

Deployment Engine — отдельный application service в `internal/core/application`; job и state machine — в domain; PostgreSQL хранит job snapshots и append-only progress events. Wails service запускает Core job асинхронно и отправляет typed `deployment:progress` только после persistence. UI восстанавливает job/event history через Go bindings. CARVION package загружается из Localization repository по project ID и сохранённому SHA-256; verifier проверяет ZIP structure и declared checksums. Единственный provider `synthetic-local-simulator` не имеет install method/capability. Production mode заблокирован константной Core policy, а не настройкой UI.

## Архитектурные пробелы

- Android XML parser отклоняет inline nested markup и CDATA внутри переводимых строк; JSON поддерживает только схему `carvion.localization/v1`.
- Localization workflow создаёт, редактирует, проверяет и экспортирует проекты; отдельный Language Packages screen и Operation History остаются placeholders.
- Simulator вызывается прямо через Wails/Core. Core выбирает доступный адаптер по capabilities, останавливает workflow при `unknown`/`unsupported` и при `supported` строит проверенный CARVION ZIP. Synthetic fixtures встроены; автомобиль не подключается. Run events существуют только в текущем результате, а project/package persistence использует Localization repository.
- Geely Research Adapter читает только выбранные пользователем файлы через RPC. Пустой exact-match catalog оставляет совместимость `unknown`; подтверждённых Geely конфигураций/build IDs для Xingyue L/Monjaro 2022–2024 нет, support не заявлен. Evidence и ограничения собраны в `docs/research/geely/`.
- Отдельных HTTP routes, migrations и direct database доступа адаптера для Simulator нет. Реальный JSON-RPC boundary проверяется отдельно от Playwright UI E2E backend.
- Desktop startup требует `DATABASE_URL` и PostgreSQL. Production Desktop storage / Hub auth не реализованы.
- `CatalogRepository` остаётся read-only и читает три исходные catalog tables. `LocalizationRepository` поддерживает транзакционную запись localization projects/resources/translations, append-only history, validation runs и package artifacts. Repositories/use cases для adapter definitions, versions, hardware variants, firmware builds и compatibility rules пока отсутствуют.
- Таблицы `adapter_definitions` и связанные справочники не синхронизированы с локальной регистрацией Manager.
- Hub-backed repository не реализован; механизм авторизации Desktop к Hub ещё требует проектирования.
- Production Deployment заблокирован: нет подтверждённой platform, exact production compatibility, разрешённого install method, production result verification/recovery procedure или bench evidence. Synthetic provider не затрагивает оборудование; настоящий rollback отсутствует. PostgreSQL deployment integration tests требуют выделенную test database.
- Отдельный процесс не является OS sandbox. Каталог адаптеров должен оставаться доверенным.
- macOS package подписан ad-hoc для локального запуска; распространение требует Developer ID signing и notarization.
- GitHub remote установлен как `https://github.com/Sneypsleep90/carvion.git`; локальная Git history начинается с публикации текущего backend.

Подробности актуального SDK API v1 и manifest schema v2 находятся в [adapters/SDK.md](adapters/SDK.md) и [adapters/PLUGIN_PROTOCOL.md](adapters/PLUGIN_PROTOCOL.md). Историческая запись исходного SDK сохранена в [adapter-sdk.md](adapter-sdk.md), снимок архитектуры до Core Refactoring — в [history/architecture-foundation-adapter-sdk.md](history/architecture-foundation-adapter-sdk.md); этот файл определяет актуальное состояние и направление.
