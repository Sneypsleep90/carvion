# История изменений

Здесь фиксируются фактические технические изменения; будущие планы приведены в [ROADMAP.md](ROADMAP.md).

## 2026-11-01 — Step 3: Adapter SDK & Plugin Manager

- Сохранён SDK API `v1`; manifest обновлён до schema v2 с `sdk_version`, `input_types` и `resource_formats`, включая строгую проверку SemVer и capability declarations.
- Plugin Manager регистрирует только локальные trusted packages с точным `GetInfo`, маршрутизирует отдельные JSON-RPC процессы по выбранному ID, проверяет форматы ресурсов, ограничивает deadline/frames/concurrency и отражает последний runtime status.
- Усилена обработка JSON-RPC: неизвестные поля, лишние JSON values/frames, malformed result, RPC error и process failure отклоняются.
- Desktop Adapters показывает SDK/API/adapter versions, capabilities, declared inputs/formats и status. Diagnostics/Localization выбирают адаптеры, совместимые с требуемой операцией и форматом.
- Обновлён manifest `adapter-android-demo`; он остаётся синтетическим и не заявляет поддержку реальных платформ.
- Добавлены SDK, protocol, adapter-authoring и security документы; фактические проверки записаны в [TESTING.md](TESTING.md).

## 2026-11-01 — Wails Desktop Application Foundation

- Добавлен отдельный nested Go module `desktop/` для Wails v3.0.0 host; root Go API module остался без Wails dependency.
- Добавлен `ApplicationService` как Wails lifecycle service: он создаёт local Core runtime, делегирует Go use cases напрямую и закрывает PostgreSQL pool при завершении.
- Добавлены generated Wails TypeScript bindings и `DesktopBackend`; React UI не вызывает REST API и не обращается к PostgreSQL.
- Создана CARVION Studio оболочка с восемью разделами, тёмной темой, responsive рабочей областью, loading/empty/error states, Tailwind CSS 4, UI primitives и Lucide.
- Реализованы Dashboard, Vehicles catalog, Diagnostics compatibility check и read-only Localization XML inspection над synthetic demo adapter. Placeholder sections явно отмечают отсутствующую функциональность.
- Настроена Apple Silicon Wails bundle сборка. Minimum macOS согласован с Go toolchain на 13.0; локальный package подписывается ad-hoc.
- Добавлены Go lifecycle service tests, Playwright E2E (3 сценария) и desktop Make targets.
- Обновлена документация состояния и команд; подробности проверок и ограничения см. в [TESTING.md](TESTING.md) и [DESKTOP.md](DESKTOP.md).

## 2026-11-01 — Подготовка публикации репозитория

- Go module и импорты переведены с временного `carvion.local` на `github.com/Sneypsleep90/carvion`.
- README и актуальные архитектурные заметки обновлены для canonical GitHub repository.
- Проверка после смены module path: `go build ./...` прошла.

## 2026-11-01 — Desktop-First Core Refactoring

- Выделены `internal/core/domain`, `internal/core/application` и `internal/core/infrastructure`.
- Добавлен `application.Service` с операциями адаптеров и чтением manufacturer/model/platform catalog через repository ports.
- Добавлены PostgreSQL read repository для существующих Foundation tables и adapter bridge к прежним SDK/Plugin Manager контрактам.
- Добавлен `internal/composition.NewLocal`; API handlers теперь вызывают Core без зависимости от SDK и Manager.
- Конфигурация Core отделена от HTTP host/port; `POSTGRES_*` остались настройками Docker Compose и не загружаются в app config.
- Новые migrations не добавлялись, существующая DB схема сохранена.
- Проверки: `make test`, `make lint`, `make build`, `make test-integration` и ручной HTTP smoke прошли; details в [TESTING.md](TESTING.md).

## 2026-11-01 — Audit & Documentation

- Добавлены документы проекта, архитектуры, стека, roadmap, решений, тестирования, проблем, прогресса и следующих задач.
- Добавлен индекс исследований адаптеров и инструкции будущим сессиям Codex в `AGENTS.md`.
- Корневой README обновлён: Desktop-first цель отделена от текущего HTTP interim state.
- Go implementation не менялась.
- Проверка: `go test -count=1 ./...` прошёл; DB integration тест не запускался без `CARVION_TEST_DATABASE_URL`.

## 2026-11-01 — Adapter SDK (исходный Step 2; новый Step 3)

- Добавлен `pkg/adaptersdk/v1` с типами и операциями JSON-RPC 2.0.
- Добавлен локальный `internal/pluginmanager` с manifest/API checks, запуском отдельного процесса, timeout и обработкой ошибок.
- Добавлен `adapter-android-demo` с synthetic JSON/XML профилями.
- Добавлена Goose migration для adapter definitions/versions, hardware variants, firmware builds и compatibility rules.
- Добавлены API routes `/api/v1/adapters` и `/api/v1/compatibility/check`.
- Реальные vehicle records не добавлялись.

## 2026-11-01 — Foundation (исходный Step 1)

- Созданы Go API и конфигурация, PostgreSQL pool, health/readiness, первая migration и локальная Compose/Make/CI основа.
