# CARVION

CARVION — Desktop-first платформа для анализа и локализации программных ресурсов автомобильных мультимедийных систем через подключаемые адаптеры. CARVION Studio использует Wails 3, React и TypeScript с прямыми Go bindings к Go Core.

В репозитории есть работающий Desktop foundation для macOS Apple Silicon, Go Core, HTTP API, PostgreSQL setup, Adapter SDK API v1 (manifest schema v2), локальный Plugin Manager и синтетический demo adapter. Localization Engine импортирует Android XML/JSON, хранит проекты/переводы, проверяет строки и собирает CARVION language packages. Реальные адаптеры отсутствуют.

Начните с [индекса документации](docs/README.md): там есть аудит, текущая и целевая архитектуры, roadmap, решения, тестовые результаты и ближайшие задачи.

Быстрый запуск Desktop, API и тестов: [RUN.md](RUN.md).

## Текущее состояние

- Go Core (`domain` / `application` / `infrastructure`), repository ports и local composition root.
- Go API поверх Core, PostgreSQL connection pool/repository и Goose migrations.
- `chi` HTTP routes для health/readiness, списка адаптеров и проверки совместимости.
- Adapter SDK API v1 / manifest v2 и JSON-RPC 2.0 over stdin/stdout; Plugin Manager валидирует и запускает локальные процессы.
- `adapter-android-demo` работает только с синтетическими тестовыми JSON/XML данными.
- Wails 3 app, React/TypeScript UI и typed bindings к Go Core.
- Desktop показывает adapter versions, capabilities, input/resource formats и последний runtime status; можно выбрать тестовый адаптер для compatibility и Android strings XML inspection.
- Localization: редактирование/утверждение переводов, validation, CSV/resource export и ZIP language package через Wails/Core.
- Simulator: контролируемые synthetic profiles проходят через отдельный JSON-RPC demo adapter, Core проверяет совместимость и тестовый localization package; доступно в Desktop.
- Отдельные Language Packages и Operation History screens, Hub-backed repository и реальные vehicle adapters пока не реализованы.

## Структура

```text
desktop/                           Wails 3 host, Go bindings, React/TypeScript Studio
cmd/carvion-api/                 API/Hub transport entry point
cmd/adapter-android-demo/        Demo process entry point
pkg/adaptersdk/v1/               Versioned SDK и JSON-RPC server
internal/core/domain/            Domain models and errors
internal/core/application/       Use cases and repository/gateway ports
internal/core/infrastructure/    PostgreSQL and adapter implementations
internal/localization/            XML/JSON/CSV engine, validation and package builder
internal/simulator/              Shared synthetic profiles and resources for tests
internal/composition/            Local runtime composition and DB lifecycle
internal/pluginmanager/          Local manifest discovery и process lifecycle
internal/adapters/androiddemo/   Synthetic demo implementation и fixtures
internal/config/                 Environment configuration
internal/database/               PostgreSQL pool
internal/server/                 Текущие HTTP routes и middleware
db/migrations/                   Goose SQL migrations
docs/README.md                   Индекс документации
docs/DESKTOP.md                  Архитектура Wails, интерфейс и запуск Studio
```

Go module использует canonical path `github.com/Sneypsleep90/carvion`. Исходный код и документация публикуются в [GitHub repository](https://github.com/Sneypsleep90/carvion).

## Локальная проверка Go API

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

## Запуск CARVION Studio на macOS

Требуются Wails CLI v3.0.0, Go 1.27.3, Node.js/npm, Xcode Command Line Tools и Docker Compose. Установите закреплённый CLI по [официальной инструкции Wails](https://v3.wails.io/quick-start/installation/): `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0`. Подготовьте `.env`, затем:

```sh
make db-up
make migrate-up
make desktop-dev
```

Production `.app`:

```sh
make desktop-build
make desktop-run
```

Подробности, ограничения режима хранения и команды тестов описаны в [docs/DESKTOP.md](docs/DESKTOP.md).

## Текущий roadmap

Следующий этап: **Step 6 — Geely Research Adapter**; перед реализацией требуется подтвердить источники и точные технические данные, поддержка не заявлена. Desktop bindings вызывают Go `application.Service` напрямую; React не использует локальный REST API. Полный план: [ROADMAP.md](docs/ROADMAP.md), ближайшие задачи: [NEXT_STEPS.md](docs/NEXT_STEPS.md).
