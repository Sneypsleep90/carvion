# Технологический стек

## Присутствует в репозитории

| Технология | Фактическое применение | Основание / границы |
|---|---|---|
| Go 1.27.3 (`go.mod`: language version 1.27.0, toolchain 1.27.3) | API, config, DB pool, SDK, Manager, demo | Основной язык существующего ядра |
| PostgreSQL 18 (`postgres:18-alpine`) | Локальная БД в Compose; Goose migrations; read-only catalog repository | Основная DB schema; Desktop dev mode использует local PostgreSQL |
| pgx/v5 5.12.0 | `pgxpool`, подключение и ping | Текущий PostgreSQL driver/pool |
| chi/v5 5.5.0 | HTTP routing | Только текущий HTTP transport; не должен попадать в core |
| Goose v3.29.0 | SQL migrations через pinned `go run` в Makefile | Не является прямой зависимостью `go.mod` |
| Docker Compose | Локальный PostgreSQL | Не запускает Desktop-приложение |
| `log/slog` | Структурированное логирование API и Manager | Стандартная библиотека Go |
| JSON-RPC 2.0 | stdin/stdout между Manager и отдельными процессами | Один request/response на вызов, frame limit и timeout |
| Внутренний Go Core | `domain`, `application`, `infrastructure`, `composition` | Транспорт-независимый Core; application ports не зависят от pgx или Plugin Manager |
| PostgreSQL repository | Чтение manufacturers, vehicle models и infotainment platforms через pgx | Local development implementation; Hub-backed repository ещё нет |
| Wails v3.0.0 | Desktop shell, embedded frontend assets и service bindings в отдельном `desktop` Go module | Зафиксирована версия framework и runtime; текущая сборка проверена на darwin/arm64 |
| React 18 + TypeScript | CARVION Studio UI и typed binding client | Strict TypeScript; Go Core вызывается напрямую, HTTP fetch отсутствует |
| Tailwind CSS 4 + shadcn/ui-style components | Дизайн-система и локальные Radix/CVA компоненты | Используются необходимые primitives в `desktop/frontend/src/components/ui` |
| Lucide React | Sidebar и интерфейсные иконки | Иконки только UI, не имеют доменной логики |
| Vite 8 + Playwright | Frontend production build и desktop workflow browser E2E | E2E исполняется на тестовом React backend; реальный Wails smoke выполняется отдельно |

Корневые Go-зависимости перечислены в `go.mod`; Wails зависимости изолированы в `desktop/go.mod`, npm packages — в `desktop/frontend/package.json` и lockfile.

## Ещё не реализовано / требует решения

| Технология | Планируемая роль | Текущий статус |
|---|---|---|
| PostgreSQL | Основная БД Desktop-продукта | Схема и pool есть; способ поставки/подключения не решён |
| Hub-backed repositories | Будущий доступ к данным сервера без DB credentials на Desktop | Port определён; удалённый клиент и auth не реализованы |

Production deployment остаётся неизвестным: текущий `.app` подписан ad-hoc для локального запуска, без Developer ID/notarization; Desktop runtime использует локальный PostgreSQL из developer environment. Wails CLI v3.0.0 и минимальную конфигурацию запуска описывает [DESKTOP.md](DESKTOP.md).
