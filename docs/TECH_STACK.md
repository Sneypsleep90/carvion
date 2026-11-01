# Технологический стек

## Присутствует в репозитории

| Технология | Фактическое применение | Основание / границы |
|---|---|---|
| Go 1.27.3 (`go.mod`: language version 1.27.0, toolchain 1.27.3) | API, config, DB pool, SDK, Manager, demo | Основной язык существующего ядра |
| PostgreSQL 18 (`postgres:18-alpine`) | Локальная БД в Compose; Goose migrations | Основная БД согласно стеку; repositories ещё отсутствуют |
| pgx/v5 5.12.0 | `pgxpool`, подключение и ping | Текущий PostgreSQL driver/pool |
| chi/v5 5.5.0 | HTTP routing | Только текущий HTTP transport; не должен попадать в core |
| Goose v3.29.0 | SQL migrations через pinned `go run` в Makefile | Не является прямой зависимостью `go.mod` |
| Docker Compose | Локальный PostgreSQL | Не запускает Desktop-приложение |
| `log/slog` | Структурированное логирование API и Manager | Стандартная библиотека Go |
| JSON-RPC 2.0 | stdin/stdout между Manager и отдельными процессами | Один request/response на вызов, frame limit и timeout |
| Внутренний Go Core | `domain`, `application`, `infrastructure`, `composition` | Транспорт-независимый Core; application ports не зависят от pgx или Plugin Manager |
| PostgreSQL repository | Чтение manufacturers, vehicle models и infotainment platforms через pgx | Local development implementation; Hub-backed repository ещё нет |

Go-зависимости перечислены в `go.mod`. Внешние npm/Node packages в репозитории не обнаружены.

## Целевой стек — ещё не подключён

| Технология | Планируемая роль | Текущий статус |
|---|---|---|
| Wails 3 | Desktop shell и Go↔UI bindings | Запланировано для Step 2 |
| React + TypeScript | Desktop UI | Нет frontend package или source tree |
| Tailwind CSS + shadcn/ui | Стили и UI components | Не установлены и не настроены |
| PostgreSQL | Основная БД Desktop-продукта | Схема и pool есть; способ поставки/подключения не решён |
| Go core | Use cases, вызываемые напрямую через Wails bindings; HTTP transport делит те же сервисы | Application service уже реализован, Wails bindings ещё нет |
| Hub-backed repositories | Будущий доступ к данным сервера без DB credentials на Desktop | Port определён; удалённый клиент и auth не реализованы |

Стек не меняется без решения. До Desktop packaging нужно исследовать управление PostgreSQL на целевых ОС, миграции и восстановление данных; конкретного ответа в репозитории нет.
