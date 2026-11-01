# Журнал прогресса

Даты указаны по дате рабочего окружения. Исторические названия Step 1/2 сохранены; новая нумерация приведена в [ROADMAP.md](ROADMAP.md).

## 2026-11-01 — Step 1: Foundation (исходная нумерация)

- **Выполнено:** Go API bootstrap, environment config, PostgreSQL pool, health/readiness routes, первая Goose migration, Make/Docker Compose setup и unit/integration test scaffolding.
- **Компоненты:** `cmd/carvion-api`, `internal/config`, `internal/database`, `internal/server`, `db/migrations/00001_create_catalog_tables.sql`, `Makefile`, Compose, CI.
- **Результаты тестов:** при завершении Foundation запускались `make test`, `make lint`, `make build` — все прошли. Полный `go test -count=1 ./...` повторно прошёл в текущем аудите (см. [TESTING.md](TESTING.md)).
- **Осталось:** application/core layer, Desktop app, DB repositories и решение по модели БД для Desktop.

## 2026-11-01 — Step 2: Adapter SDK (исходная нумерация; новый Step 3)

- **Выполнено:** SDK v1, JSON-RPC server/client, manifest validation и local discovery, process timeout/cleanup, compatibility schema, demo adapter на synthetic данных, два HTTP route и документация.
- **Компоненты:** `pkg/adaptersdk/v1`, `internal/pluginmanager`, `internal/adapters/androiddemo`, `cmd/adapter-android-demo`, `internal/server`, `db/migrations/00002_adapter_sdk.sql`.
- **Результаты тестов:** `make test`, `make lint`, `make build`, `go test -race ./internal/pluginmanager ./pkg/adaptersdk/v1` прошли. Миграция проходила Up/Down/Up; API smoke включал health/readiness, список adapters, точное synthetic совпадение и ответы `unknown`.
- **Осталось:** отделить Desktop core от HTTP и подключить adapter operations через Wails/core; реальные platform claims требуют исследования.

## 2026-11-01 — Step 0: Project Audit & Documentation

- **Выполнено:** инвентаризированы структура и зависимости, проверены HTTP boundaries, конфигурация, migrations, SDK/Manager, tests и Desktop-first пробелы; подготовлены документы и `AGENTS.md`. Реализация не удалялась и не переписывалась.
- **Компоненты:** корневые `README.md`, `AGENTS.md`, `docs/`.
- **Результаты тестов:** `go test -count=1 ./...` прошёл для всех unit-test пакетов. Integration-tagged тест не запускался: `CARVION_TEST_DATABASE_URL` не задан.
- **Осталось:** выполнить Step 1 Core Refactoring; задачи описаны в [NEXT_STEPS.md](NEXT_STEPS.md).

## 2026-11-01 — Step 1: Desktop-First Core Refactoring

- **Выполнено:** созданы domain/application/infrastructure слои, application service и порты `AdapterGateway`/`CatalogRepository`; добавлены local PostgreSQL repository и мост к существующему Plugin Manager/SDK. `internal/composition.NewLocal` собирает runtime и закрывает DB pool; HTTP config выделена отдельно. API handlers используют Core/domain, маршруты и внешние ответы сохранены. Adapter SDK, Demo Adapter и migrations не переписывались.
- **Компоненты:** `internal/core/domain`, `internal/core/application`, `internal/core/infrastructure/{postgres,adapterbridge}`, `internal/composition`, `internal/config`, `internal/server`, `cmd/carvion-api`, новые tests.
- **Результаты тестов/проверок:** `make test`, `make lint`, `make build`, `make test-integration` прошли. Migration version осталась 2. PostgreSQL integration test прочитал все три catalog tables. API smoke: `/healthz`, `/readyz`, adapter list и exact synthetic compatibility вернули ожидаемые ответы. После SIGINT HTTP server выполнил graceful shutdown.
- **Осталось:** Step 2 Wails shell/UI и Go binding facade; Hub-backed remote repository и его auth не реализованы. Repository пока read-only и читает только `manufacturers`, `vehicle_models`, `infotainment_platforms`; дальнейшая работа указана в [NEXT_STEPS.md](NEXT_STEPS.md).
