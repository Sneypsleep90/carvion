# Roadmap CARVION

Порядок ниже заменяет прежнюю последовательность, где SDK шёл сразу после Foundation. Фактически уже написанный SDK/Plugin Manager сохраняется и интегрируется в новый Step 3; исторические записи в [PROGRESS.md](PROGRESS.md) не перенумеровываются.

| Этап | Название | Статус аудита |
|---|---|---|
| Step 0 | Audit & Documentation | Выполнено 2026-11-01; код не менялся |
| Step 1 | Desktop-First Core Refactoring | Выполнено 2026-11-01; Core вызывает Go-сервисы напрямую, HTTP сохранён отдельным transport |
| Step 2 | Wails Desktop Foundation | Выполнено 2026-11-01; Wails app, React UI, direct bindings и macOS packaging |
| Step 3 | Adapter SDK & Plugin Manager | Выполнено 2026-11-01; manifest v2, SDK contract, process RPC и Desktop adapter selection |
| Step 4 | Localization Engine | Следующий этап |
| Step 5 | CARVION Simulator | Запланировано; синтетические сценарии должны быть явно отделены от реальных систем |
| Step 6 | Geely Research Adapter | Запланировано; источники и данные требуют исследования, поддержка не заявлена |
| Step 7 | Deployment Engine | Запланировано; только разрешённые и документированные операции |
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

### Step 4 — Localization Engine

Построить workflow анализа ресурсов, каталога переводов, review и экспорта. Определить формат проекта, историю и проверяемость изменений. Использовать явный вход пользователя или разрешённый источник; deployment сюда не включать.

### Step 5 — CARVION Simulator

Развить изолированный симулятор для сценариев без настоящего автомобиля. Маркировать сценарии и данные как synthetic и проверять их автоматическими тестами.

### Step 6 — Geely Research Adapter

Сначала документировать источники и допустимые способы анализа. Создавать адаптер после подтверждения платформы, аппаратных ревизий, точных build identifiers и правомерности данных. До достаточных доказательств результат остаётся `unknown`; совместимость Geely не заявляется.

### Step 7 — Deployment Engine

Определить разрешённый deployment scope, проверку целостности, явное подтверждение пользователя, безопасное прерывание и rollback для документированных процедур. Не реализовывать обход защиты или несанкционированную запись в автомобиль.

### Step 8 — CARVION Hub & Commercial Platform

Проработать происхождение и проверку подписанных дистрибутивов адаптеров, обновления, аккаунты и коммерческую модель. Не добавлять сетевую доставку кода до threat model и решения о доверии.

Детализация ближайших задач: [NEXT_STEPS.md](NEXT_STEPS.md). Все дальнейшие этапы являются планом, кроме явно отмеченных выполненными.
