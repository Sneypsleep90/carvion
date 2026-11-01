# CARVION

CARVION — Desktop-first платформа для анализа и локализации программных ресурсов автомобильных мультимедийных систем через подключаемые адаптеры. Целевой интерфейс — Desktop-приложение на Wails 3, React и TypeScript с Go-ядром.

В текущем репозитории реализована ранняя Go-основа: HTTP API, PostgreSQL setup, Adapter SDK v1, локальный Plugin Manager и синтетический demo adapter. Desktop-приложения, localization workflow и реальных адаптеров пока нет.

Начните с [индекса документации](docs/README.md): там есть аудит, текущая и целевая архитектуры, roadmap, решения, тестовые результаты и ближайшие задачи.

## Текущее состояние

- Go Core (`domain` / `application` / `infrastructure`), repository ports и local composition root.
- Go API поверх Core, PostgreSQL connection pool/repository и Goose migrations.
- `chi` HTTP routes для health/readiness, списка адаптеров и проверки совместимости.
- Adapter SDK v1 и JSON-RPC 2.0 over stdin/stdout; Plugin Manager запускает локальные процессы.
- `adapter-android-demo` работает только с синтетическими тестовыми JSON/XML данными.
- Wails, React UI, localization engine, simulator, Hub-backed repository и реальные vehicle adapters не реализованы.

## Структура

```text
cmd/carvion-api/                 API/Hub transport entry point
cmd/adapter-android-demo/        Demo process entry point
pkg/adaptersdk/v1/               Versioned SDK и JSON-RPC server
internal/core/domain/            Domain models and errors
internal/core/application/       Use cases and repository/gateway ports
internal/core/infrastructure/    PostgreSQL and adapter implementations
internal/composition/            Local runtime composition and DB lifecycle
internal/pluginmanager/          Local manifest discovery и process lifecycle
internal/adapters/androiddemo/   Synthetic demo implementation и fixtures
internal/config/                 Environment configuration
internal/database/               PostgreSQL pool
internal/server/                 Текущие HTTP routes и middleware
db/migrations/                   Goose SQL migrations
docs/README.md                   Индекс документации
```

Go module использует canonical path `github.com/Sneypsleep90/carvion`. Исходный код и документация публикуются в [GitHub repository](https://github.com/Sneypsleep90/carvion).

## Локальная проверка текущей основы

Требуются Go 1.27.3, Docker Compose и `make`.

```sh
cp .env.example .env
make build
make db-up
make migrate-up
make dev
```

API smoke commands:

```sh
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl http://localhost:8080/api/v1/adapters
curl -X POST http://localhost:8080/api/v1/compatibility/check \
  -H 'Content-Type: application/json' \
  -d '{"adapter_id":"adapter-android-demo","identity":{"manufacturer":"CARVION TEST LAB","model":"Android Demo Unit","platform_id":"android-demo-platform"},"hardware":{"platform_id":"android-demo-platform","revision":"rev-a"},"firmware":{"build_id":"demo-build-a-001","version":"1.0.0-demo"}}'
```

Unit checks не требуют PostgreSQL:

```sh
make test
make lint
make build
```

Integration test подключается только к явно заданной `CARVION_TEST_DATABASE_URL`; см. [TESTING.md](docs/TESTING.md). `ADAPTERS_DIR` по умолчанию `adapters`. Только доверенные локальные процессы должны находиться в каталоге.

## Текущий roadmap

Следующий этап: **Step 2 — Wails Desktop Foundation**. Wails bindings должны вызывать `application.Service` напрямую; Desktop не должен обращаться к Core через localhost REST API. SDK и Plugin Manager уже существуют и подключены к Core через infrastructure bridge. Полный план: [ROADMAP.md](docs/ROADMAP.md), ближайшие задачи: [NEXT_STEPS.md](docs/NEXT_STEPS.md).
