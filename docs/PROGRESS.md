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

## 2026-11-01 — Step 2: Desktop Application Foundation

- **Выполнено:** добавлен Wails 3.0.0 Desktop host и embedded React/TypeScript UI. Typed Wails `ApplicationService` создаёт local runtime через `composition.NewLocal`, вызывает Go Core напрямую, передаёт lifecycle context и закрывает runtime. Добавлены восемь разделов Studio; live dashboard, catalog, synthetic compatibility и read-only Android XML inspection используют реальные Core/adapter calls. UI отображает unavailable/degraded/errors и явно говорит, что автомобиль не подключён. Добавлены Tailwind CSS 4, shadcn/ui-style primitives и Lucide icons. Создана arm64 `.app`; minimum macOS выставлен на 13.0 после предупреждения toolchain о несовпадении с 12.0.
- **Компоненты:** `desktop/` Wails module, `desktop/application_service.go` и lifecycle tests, generated bindings и `desktop/frontend/`; `Makefile`; корневой `README.md`; `docs/DESKTOP.md`, architecture/roadmap/project/stack/adapter/decision/testing/progress/changelog/issues/next steps.
- **Результаты тестов/проверок:** `make desktop-test` прошёл (root и Wails Go tests, `tsc`, 3 Playwright E2E tests, Vite production build); `make lint` прошёл; `make build` прошёл; `make desktop-build` создал arm64 bundle и проверил ad-hoc codesign; `make desktop-run` открыл Studio. Native smoke подтвердил Core ready, PostgreSQL connected, 1 synthetic adapter, exact compatibility и 2 переводимые XML строки. Локальная DB была на migration version 2. Детали и ограничения записаны в [TESTING.md](TESTING.md).
- **Осталось:** Step 3 — расширенная приёмка SDK/Manager flows в UI, RPC timeout/error presentation и решение по adapter registry synchronization. Desktop пока development mode на local PostgreSQL; Hub mode/auth, localization editing/export, persistent history и production signing/notarization не реализованы.

## 2026-11-01 — Step 3: Adapter SDK & Plugin Manager

- **Выполнено:** сохранён SDK API v1 и добавлен manifest schema v2 с SDK SemVer, `input_types`, `resource_formats` и строгой проверкой capability consistency. Plugin Manager обнаруживает trusted local packages, валидирует manifest/executable/API/SDK, запускает отдельный процесс на RPC вызов, проверяет envelope/result, формат ресурса, ошибки, timeout и cancellation; status отражает последний вызов. `adapter-android-demo` объявляет новые metadata. Desktop показывает версии, capabilities, входы, форматы и status; Diagnostics/Localization вызывают выбранный адаптер по capability и формату. Добавлены тесты маршрутизации между двумя процессами, malformed manifest/response, SDK mismatch, RPC error, timeout, unknown profile, XML и UI selection.
- **Компоненты:** `pkg/adaptersdk/v1`, `internal/pluginmanager`, `internal/core/domain`, `internal/core/infrastructure/adapterbridge`, `internal/server`, `internal/adapters/androiddemo`, `adapters/adapter-android-demo`, `desktop/frontend`, `docs/adapters/`.
- **Результаты тестов:** `make desktop-test` прошёл (root/nested Go tests, TypeScript, 4 Playwright E2E и frontend production build); `go test -race ./internal/pluginmanager ./pkg/adaptersdk/v1`, `make lint`, `make build`, `make desktop-build` прошли. Упакованный app запустился с Core ready, PostgreSQL connected и зарегистрированным demo adapter; native Wails compatibility вернула `supported` для точного synthetic profile, XML inspection вернула 2 переводимые строки. Детали и ограничения см. в [TESTING.md](TESTING.md).
- **Осталось:** Step 4 — Localization Engine. Package signing, OS sandbox, remote registry, DB adapter-table synchronization, real platform profiles и firmware deployment не реализованы.

## 2026-11-01 — Step 4: Localization Engine

- **Выполнено:** добавлены Android XML (`string`, `string-array`, `plurals`), JSON schema `carvion.localization/v1`, строгий UTF-8 и CSV import/export. Core поддерживает проекты, сохранение/утверждение переводов, повторный импорт source с переводом изменённых утверждений в `needs_review`, validation report, экспорт ресурсов и детерминированный ZIP language package с manifest, validation report и SHA-256. Self-closing XML exports разворачиваются в валидные парные элементы, namespaced корни и `string`/`item` resources отклоняются; history сохраняет hash source предыдущего approval. PostgreSQL хранит проекты, ресурсы, переводы, append-only history, validation runs и package bytes. Localization UI работает через Wails bindings напрямую к Go Core; HTTP routes не добавлялись.
- **Компоненты:** `internal/localization`, `internal/core/domain/localization.go`, `internal/core/application/localization.go`, `internal/core/infrastructure/postgres/localization_repository.go`, `db/migrations/00003_localization_engine.sql`, `internal/composition`, `desktop/application_service.go`, generated Wails bindings и `desktop/frontend`.
- **Результаты тестов:** после review fix `make desktop-test` прошёл (root/Desktop Go, TypeScript, 5 Playwright E2E и Vite build); `go test -tags=integration -count=1 ./...` прошёл на отдельной базе после Goose migration version 3; `make test-integration`, `make lint`, `make build`, `go test -race ./internal/localization ./internal/core/application` и `make desktop-build` прошли. Wails собрал production `.app` с успешной ad-hoc codesign verification; native UI smoke повторно не выполнялся. Подробности приведены в [TESTING.md](TESTING.md).
- **Осталось:** nested markup/CDATA в Android strings не поддерживаются; `Language Packages` и `Operation History` как отдельные экраны остаются placeholder. Следующий этап — Step 5 Simulator. Hub storage/auth, реальные автомобильные данные и firmware deployment не реализованы.

## 2026-11-01 — Step 5: CARVION Simulator

- **Выполнено:** добавлен единый встроенный каталог синтетических Platform A/B, unknown firmware, incompatible hardware и контролируемой RPC ошибки; `adapter-android-demo` использует те же fixtures. Core запускает ограниченный контекстом сценарий, выбирает адаптер по capabilities, сверяет identity и точную compatibility, прекращает workflow на `unknown`/`unsupported`, а для `supported` импортирует demo XML, выполняет тестовую локализацию, валидацию и проверяет SHA-256/ZIP/checksums/non-OEM manifest. Wails Simulator экран вызывает Core напрямую, показывает события, отмену и загрузку пакета.
- **Компоненты:** `internal/simulator`, `internal/core/{domain,application}`, `internal/adapters/androiddemo`, `desktop/ApplicationService`, `desktop/frontend` и E2E, Wails bindings, `docs/simulator/`; миграция не добавлялась, HTTP API не менялся.
- **Результаты тестов:** `make desktop-test` прошёл (root/Desktop Go, TypeScript, 10 Playwright E2E, production frontend build); `go test -tags=integration -count=1 ./...` прошёл после Goose migrations 1–3 на выделенной временной PostgreSQL базе, включая реальный JSON-RPC subprocess и persistence тест; `go test -race -count=1 ./internal/simulator ./internal/adapters/androiddemo ./internal/core/application`, Desktop race suite, `make lint`, `make build`, `make desktop-build` и `git diff --check` прошли.
- **Осталось:** только synthetic данные; нет native Wails UI automation, persistent Simulator run history, настоящих автомобильных адаптеров или установки. Истечение общего двухминутного run deadline отдельно не тестировалось. Следующий этап — Step 6 research по Geely; сначала требуется подтверждённое evidence, support не заявлен.
