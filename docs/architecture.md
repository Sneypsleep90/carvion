# Архитектура CARVION

Текущий код отделён здесь от целевой архитектуры. Порядок этапов указан в [ROADMAP.md](ROADMAP.md).

## Текущая архитектура — реализовано

```text
cmd/carvion-api (API/Hub transport)
  ├── config.Load + config.LoadHTTP
  ├── composition.NewLocal
  │     ├── application.Service
  │     ├── infrastructure/postgres.CatalogRepository ─── PostgreSQL
  │     └── infrastructure/adapterbridge ── Plugin Manager
  └── internal/server (chi handlers) ── application.Service

application.Service
  ├── domain types and errors
  ├── CatalogRepository port ── local PostgreSQL implementation
  └── AdapterGateway port ── SDK/Plugin Manager bridge
                                  └── JSON-RPC process → adapter-android-demo

Potential Wails entry point → application.Service directly (not implemented)
```

`internal/core/domain` содержит transport-neutral типы и ошибки; `internal/core/application` содержит use cases и порты; `internal/core/infrastructure` реализует PostgreSQL catalog и переводит контракты SDK/Manager в Core domain. `internal/composition.NewLocal` создаёт pool, локальный adapter manager, infrastructure и application service и владеет закрытием PostgreSQL pool. API запускает этот Core в процессе, HTTP handlers вызывают его напрямую.

Wails и binding facade пока не добавлены. Когда они появятся, Go binding будет вызывать `application.Service` напрямую; локальный HTTP API для Desktop не требуется.

## Инвентаризация репозитория

| Путь | Назначение / наблюдение |
|---|---|
| `cmd/` | Две Go entry points: API и синтетический adapter process |
| `pkg/adaptersdk/v1/` | Публичный Go contract и JSON-RPC server |
| `internal/core/domain/` | Domain types и ошибки |
| `internal/core/application/` | Application service и порты `AdapterGateway` / `CatalogRepository` |
| `internal/core/infrastructure/` | PostgreSQL catalog repository и adapter bridge к SDK/Manager |
| `internal/composition/` | Создание local runtime и lifecycle PostgreSQL pool |
| `internal/` | Также config, DB pool, HTTP transport, Plugin Manager, demo adapter и тесты |
| `adapters/` | Demo `manifest.json`; `bin/` содержит generated executable после build |
| `db/migrations/` | Две Goose migrations |
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
| `internal/composition` | Собирает local PostgreSQL runtime и adapters; общая точка для Hub API и будущего Desktop launcher |

HTTP API остаётся отдельным transport и текущей точкой входа для Hub/API процесса. Существующие маршруты сохранены. По HTTP доступны список адаптеров и проверка совместимости; `Detect` и `InspectResources` доступны через Core application service, отдельные HTTP routes для них пока отсутствуют. Desktop будет обращаться к тем же сервисам напрямую из Go.

`CatalogRepository` реализован для чтения `manufacturers`, `vehicle_models`, `infotainment_platforms`. Остальные таблицы Step 2 и миграции сохранены, но CRUD и синхронизация с локальным manifest catalog не реализованы. Будущий remote/Hub-backed repository должен реализовать тот же application port; такая реализация пока отсутствует.

## Целевая Desktop-first архитектура — запланировано

```text
React + TypeScript UI
        │ Wails binding facade
        ▼
Wails 3 Desktop composition root
        │ прямой Go-вызов
        ▼
Go application/core services ───────────── Hub HTTP transport
  ├── domain / use cases / errors            (тот же Core)
  ├── repository port ── local PG (dev)     (отдельный процесс Hub)
  │                └── Hub-backed client     └── repository/server services
  └── adapter port → Plugin Manager
                       └── JSON-RPC process → adapter
```

Wails bindings и HTTP handlers вызывают application services; Core не вызывается через локальный REST endpoint. Для разработки локальный composition root использует PostgreSQL. Application зависит от repository ports, поэтому Hub-backed implementation можно подключить без изменений use cases и без административных DB credentials в Desktop. Remote implementation и авторизация Hub ещё не реализованы. Plugin Manager остаётся границей Core и adapter processes.

## Архитектурные пробелы

- Нет Wails app или binding facade; Core подготовлен к прямому Go-вызову, но ещё не подключён к Desktop UI.
- PostgreSQL repository read-only и пока покрывает только три исходные catalog tables; adapter definitions, versions, hardware variants, firmware builds и compatibility rules доступны только через схему миграций.
- Таблицы `adapter_definitions` и связанные справочники не синхронизированы с локальной регистрацией Manager.
- Hub-backed repository не реализован; механизм авторизации Desktop к Hub ещё требует проектирования.
- Отдельный процесс не является OS sandbox. Каталог адаптеров должен оставаться доверенным.
- GitHub remote установлен как `https://github.com/Sneypsleep90/carvion.git`; локальная Git history начинается с публикации текущего backend.

Подробности SDK v1 находятся в [adapter-sdk.md](adapter-sdk.md). Исторический снимок архитектуры до Core Refactoring сохранён в [history/architecture-foundation-adapter-sdk.md](history/architecture-foundation-adapter-sdk.md); этот файл определяет актуальное состояние и направление.
