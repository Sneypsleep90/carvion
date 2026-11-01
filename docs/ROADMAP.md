# Roadmap CARVION

Порядок ниже заменяет прежнюю последовательность, где SDK шёл сразу после Foundation. Фактически уже написанный SDK/Plugin Manager сохраняется и интегрируется в новый Step 3; исторические записи в [PROGRESS.md](PROGRESS.md) не перенумеровываются.

| Этап | Название | Статус аудита |
|---|---|---|
| Step 0 | Audit & Documentation | Выполнено 2026-11-01; код не менялся |
| Step 1 | Desktop-First Core Refactoring | Выполнено 2026-11-01; Core вызывает Go-сервисы напрямую, HTTP сохранён отдельным transport |
| Step 2 | Wails Desktop Foundation | Следующий этап; Wails/UI пока отсутствуют |
| Step 3 | Adapter SDK & Plugin Manager | Основные SDK/manager/demo реализованы и тестировались ранее; требуется интеграция с новым core и приёмка по новому порядку |
| Step 4 | Localization Engine | Запланировано |
| Step 5 | CARVION Simulator | Запланировано; синтетические сценарии должны быть явно отделены от реальных систем |
| Step 6 | Geely Research Adapter | Запланировано; источники и данные требуют исследования, поддержка не заявлена |
| Step 7 | Deployment Engine | Запланировано; только разрешённые и документированные операции |
| Step 8 | CARVION Hub & Commercial Platform | Запланировано |

## Цели и критерии готовности этапов

### Step 0 — Audit & Documentation

Зафиксировать структуру, текущие связи, test evidence, риски и Desktop-first очередь. Готово, когда документация синхронизирована с кодом, тесты запущены, а неподтверждённые функции отмечены как план или исследование.

### Step 1 — Desktop-First Core Refactoring — выполнено 2026-11-01

Выделены `internal/core/domain`, `internal/core/application` и `internal/core/infrastructure`; добавлены application ports, PostgreSQL catalog repository и adapter bridge. `internal/composition.NewLocal` собирает Core и управляет DB lifecycle. HTTP API сохранён как отдельный transport, его handlers вызывают Core; конфигурация HTTP отделена от Core config. SDK, Manager, Demo Adapter и migrations сохранены. Проверки: unit/service tests, integration test PostgreSQL repository, API smoke и graceful shutdown. Wails пока не добавлен; режим будущего Hub-backed repository и авторизация требуют следующих решений.

### Step 2 — Wails Desktop Foundation

Добавить Wails 3 shell, React/TypeScript UI и выбранную настройку Tailwind/shadcn/ui; связать UI только с Go application services через bindings. Готово, когда Desktop запускается на согласованных ОС, показывает базовое состояние приложения, а UI не содержит доменную логику.

### Step 3 — Adapter SDK & Plugin Manager

Сохранить и интегрировать существующие SDK v1, manifest discovery, process JSON-RPC и demo в новые core services. Закрыть пробелы контрактов для вызовов адаптеров из Desktop. Готово, когда Desktop core вызывает синтетический адаптер и корректно представляет app-level results/errors.

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

Детализация ближайших задач: [NEXT_STEPS.md](NEXT_STEPS.md). Все будущие стадии являются планом, за исключением уже реализованных компонентов Step 3.
