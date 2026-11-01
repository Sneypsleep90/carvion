# Roadmap CARVION

Порядок ниже заменяет прежнюю последовательность, где SDK шёл сразу после Foundation. Фактически уже написанный SDK/Plugin Manager сохраняется и интегрируется в новый Step 3; исторические записи в [PROGRESS.md](PROGRESS.md) не перенумеровываются.

| Этап | Название | Статус аудита |
|---|---|---|
| Step 0 | Audit & Documentation | Выполнено 2026-11-01; код не менялся |
| Step 1 | Desktop-First Core Refactoring | Выполнено 2026-11-01; Core вызывает Go-сервисы напрямую, HTTP сохранён отдельным transport |
| Step 2 | Wails Desktop Foundation | Выполнено 2026-11-01; Wails app, React UI, direct bindings и macOS packaging |
| Step 3 | Adapter SDK & Plugin Manager | Выполнено 2026-11-01; manifest v2, SDK contract, process RPC и Desktop adapter selection |
| Step 4 | Localization Engine | Выполнено 2026-11-01; детали в `docs/localization/` |
| Step 5 | CARVION Simulator | Выполнено 2026-11-01; детали и реальные результаты — `docs/simulator/` |
| Step 6 | Geely Research Adapter | Исследовательский каркас реализован; точные конфигурации неизвестны, поддержка не заявлена |
| Step 7 | Deployment Engine | Реализован Core, PostgreSQL audit, synthetic provider и Desktop workspace; Production Mode заблокирован, аппаратных тестов нет |
| Step 8 | CARVION Hub & Commercial Platform | Запланировано |

## Цели и критерии готовности этапов

### Step 0 — Audit & Documentation

Зафиксировать структуру, текущие связи, test evidence, риски и Desktop-first очередь. Готово, когда документация синхронизирована с кодом, тесты запущены, а неподтверждённые функции отмечены как план или исследование.

### Step 1 — Desktop-First Core Refactoring — выполнено 2026-11-01

Выделены `internal/core/domain`, `internal/core/application` и `internal/core/infrastructure`; добавлены application ports, PostgreSQL catalog repository и adapter bridge. `internal/composition.NewLocal` собирает Core и управляет DB lifecycle. HTTP API сохранён как отдельный transport, его handlers вызывают Core; конфигурация HTTP отделена от Core config. SDK, Manager, Demo Adapter и migrations сохранены. Проверки: unit/service tests, integration test PostgreSQL repository, API smoke и graceful shutdown. Wails пока не добавлен; режим будущего Hub-backed repository и авторизация требуют следующих решений.

### Step 2 — Wails Desktop Foundation — выполнено 2026-11-01

Добавлен отдельный Go module `desktop/` с Wails v3.0.0, embedded React/TypeScript UI, Tailwind CSS 4, shadcn-style UI primitives и Lucide icons. `ApplicationService` создаёт `composition.NewLocal`, использует Wails lifecycle context, напрямую вызывает Core и закрывает runtime при shutdown. UI показывает live local Core/PostgreSQL/adapter state, каталог, synthetic compatibility и Android XML inspection; REST не используется для UI. Разделы языковых пакетов/истории и часть настроек пока placeholders. `make desktop-test`, lint и builds прошли; native arm64 `.app` запущен, compatibility/XML operations проверены через Wails bindings. Подпись/notarization и production storage остаются за рамками.

### Step 3 — Adapter SDK & Plugin Manager — выполнено 2026-11-01

SDK API v1 сохранён; manifest schema v2 объявляет SDK SemVer, входные типы и форматы ресурсов. Manager валидирует manifest, обнаруживает trusted local packages, регистрирует только процесс с совпадающим `GetInfo`, выполняет JSON-RPC 2.0 через stdin/stdout и обрабатывает malformed responses, errors, parent cancellation и timeout. `adapter-android-demo` остаётся синтетическим. Desktop показывает metadata и последний runtime status; Diagnostics и Localization вызывают выбранный адаптер только при наличии нужной capability/формата. Тесты покрывают SDK/API mismatch, process launch/routing, timeout, ошибки RPC и UI selection. Процессы не являются OS sandbox; Hub distribution, package signing и реальные автомобильные данные не реализованы.

### Step 4 — Localization Engine — выполнено 2026-11-01

Добавлены Android XML (`string`, `string-array`, `plurals`), JSON `carvion.localization/v1`, CSV import/export, проекты локализации, переводы `draft`/`approved`/`needs_review`, проверка placeholders/plurals/UTF-8 и source SHA-256 review. PostgreSQL хранит проекты, ресурсы, переводы, append-only историю, validation runs и ZIP artifacts. Wails Localization screen напрямую вызывает Go Core для импорта, редактирования, review, экспорта и package build. Package помечен как CARVION localization (`oem_firmware=false`). HTTP endpoints для Desktop workflow не добавлялись. Nested XML markup и CDATA внутри переводимых строк пока отклоняются. См. [localization/ENGINE.md](localization/ENGINE.md).

### Step 5 — CARVION Simulator — выполнено 2026-11-01

Добавлен единый локальный каталог synthetic profiles и ресурсов; Demo Adapter использует его в отдельном JSON-RPC процессе. Core выбирает адаптер по capabilities, повторно сверяет точную identity, hardware и firmware, останавливает unknown/unsupported до localization, а для поддержанного тестового профиля импортирует ресурсы, валидирует русские переводы и проверяет ZIP, SHA-256 и `oem_firmware=false`. Wails Simulator screen вызывает Core напрямую, показывает события, умеет отменять сценарий и скачивать пакет. Проверки включают unit/service, E2E, реальный subprocess RPC и PostgreSQL integration на изолированной базе. Профили и package — только synthetic; подключения к автомобилю и установки нет. Подробности: [simulator/OVERVIEW.md](simulator/OVERVIEW.md), [simulator/TEST_RESULTS.md](simulator/TEST_RESULTS.md).

### Step 6 — Geely Research Adapter — исследовательский каркас реализован 2026-11-01

Добавлены локальный process adapter, строгий versioned JSON profile, официальный evidence catalog с областью применимости, пустой exact compatibility catalog, XML/JSON resource analysis через существующий parser и Wails Geely Research Mode. Localization import сохраняет пользовательский файл только после сверки Core и adapter parser; диагностический отчёт не содержит resource content или filesystem path. Это не завершённое исследование аппаратуры: OS/build IDs и аппаратные ревизии Xingyue L / Monjaro 2022–2024 не установлены. В отсутствие полного проверенного правила статус `unknown`, поддержка Geely не заявлена. PostgreSQL schema/migration и HTTP routes для Desktop не добавлялись. См. [docs/research/geely/OVERVIEW.md](research/geely/OVERVIEW.md).

### Step 7 — Deployment Engine — тестовый scope реализован 2026-11-02

Добавлены persistent Deployment Jobs/events, package integrity preflight, строгая state machine, result/recovery capability checks, Wails progress и Desktop history. Подключён только synthetic provider без возможности установки; Production Mode всегда заблокирован Core policy из-за отсутствия проверенных platform/install/compatibility/recovery/bench evidence. Автомобильные и стендовые проверки не выполнялись. См. [deployment/](deployment/ARCHITECTURE.md).

### Step 8 — CARVION Hub & Commercial Platform

Проработать происхождение и проверку подписанных дистрибутивов адаптеров, обновления, аккаунты и коммерческую модель. Не добавлять сетевую доставку кода до threat model и решения о доверии.

Детализация ближайших задач: [NEXT_STEPS.md](NEXT_STEPS.md). Все дальнейшие этапы являются планом, кроме явно отмеченных выполненными.
