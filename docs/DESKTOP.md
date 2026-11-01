# CARVION Studio — Desktop Foundation

CARVION Studio — macOS Desktop приложение для работы с локальным Go Core. Desktop UI не связывается с Core по HTTP: React использует сгенерированные Wails bindings к Go service facade.

## Состояние

**Реализовано:** Wails 3.0.0 shell; React 18 + TypeScript; Tailwind CSS 4 и локальные shadcn/ui-style components; Lucide icons; typed Go service facade; embedded production assets; локальный runtime через `composition.NewLocal`; навигация по восьми разделам.

**Рабочие workflows:** состояние Core/PostgreSQL/адаптеров, синтетический adapter registry, каталог из PostgreSQL, synthetic compatibility checks, Localization Engine для локальных файлов Android XML/JSON и Simulator для локальных synthetic профилей.

**Placeholder / запланировано:** отдельные Language Packages и Operation History screens и часть Settings показывают будущие состояния. Нет deployment, реального подключения к автомобилю, Hub client или Hub authentication.

## Архитектура

```text
desktop/frontend React UI
        │ generated TypeScript bindings
        ▼
desktop.ApplicationService (Wails service + lifecycle)
        │ direct Go calls, Wails context
        ▼
internal/composition.NewLocal
        ├── application.Service (catalog + localization use cases)
        ├── local PostgreSQL catalog/localization repositories
        ├── internal/localization (XML/JSON/CSV, validation, exports, ZIP)
        └── Plugin Manager → local JSON-RPC adapter process
```

`desktop/` — отдельный Go module с `replace` на корневой module. Это удерживает Wails-native dependencies вне корневого Go API module. `desktop/main.go` встраивает `frontend/dist` и регистрирует `ApplicationService` в Wails. Для текущего dev profile facade загружает конфигурацию и создаёт Core в `ServiceStartup`; при ошибке показывает degraded/unavailable state, чтобы UI мог сообщить причину. `ServiceShutdown` закрывает local runtime.

Wails service binding API проверен для закреплённой версии `v3.0.0`: Wails регистрирует типизированные Go services через `application.NewService`, генерирует TypeScript bindings, вызывает `ServiceStartup(ctx, options)` и `ServiceShutdown()`. Официальные материалы: [Wails v3.0.0 release](https://github.com/wailsapp/wails/releases), [Services & bindings](https://v3.wails.io/features/bindings/services/), [Lifecycle](https://v3.wails.io/concepts/lifecycle/), [First app](https://v3.wails.io/quick-start/first-app/).

Generated bindings находятся в `desktop/frontend/bindings/github.com/Sneypsleep90/carvion/desktop`. `desktop/frontend/src/lib/backend.ts` является UI-facing typed interface; production implementation импортирует generated `ApplicationService` и не использует fetch/REST. `e2e-entry.tsx` отдельно подменяет backend только в Playwright route.

## Интерфейс

Темная инженерная рабочая среда содержит sidebar, topbar, breadcrumbs, статус локального runtime и main workspace. Общие цвета и компоненты задаются в `desktop/frontend/src/index.css` и `src/components/ui/`.

| Раздел | Реальное поведение сейчас |
|---|---|
| Dashboard | Live состояние Core, PostgreSQL и количество обнаруженных адаптеров; переходы к основным задачам |
| Vehicles | Read-only manufacturer/model/platform каталог из Core/PostgreSQL |
| Diagnostics | Ручной synthetic vehicle profile и `supported` / `unsupported` / `unknown` compatibility result |
| Localization | Импорт Android XML/JSON, проекты и русские переводы, поиск, edit/approve, reimport source, validation, CSV/resource export и ZIP package download. Прямой Wails-to-Core вызов |
| Simulator | Выбор synthetic профиля, запуск Core workflow через JSON-RPC demo process, event log, отмена, compatibility/package verification и загрузка тестового ZIP. Не подключается к автомобилю |
| Language Packages | Placeholder отдельного экрана; сборка и скачивание доступны на экране Localization |
| Adapters | Адаптеры из доверенного локального каталога и доступность Core |
| Operation History | Placeholder; операции пока не сохраняются в журнал |
| Settings | Текущий режим/состояние; полноценные настройки пользователя не сохраняются |

UI показывает loading, empty и error states. При неизвестном compatibility profile интерфейс показывает `unknown`. Приложение не сканирует автомобиль и не изображает успешное подключение к реальному vehicle.

## Требования и запуск

Проверенная среда: macOS 26.6.2 darwin/arm64, Go 1.27.3, Node.js 26.3/npm 11.16, Docker Compose, Xcode Command Line Tools и Wails CLI v3.0.0. Создаваемая сборка Apple Silicon (`arm64`) объявляет minimum macOS 13.0. Перед локальным запуском установите CLI по [официальной инструкции](https://v3.wails.io/quick-start/installation/), подготовьте `.env` по `.env.example` и запустите PostgreSQL:

```sh
make db-up
make migrate-up
make desktop-dev
```

`desktop-dev` собирает `adapter-android-demo`, устанавливает `ADAPTERS_DIR` на локальный каталог и запускает Wails dev workflow. Core читает `DATABASE_URL` из `.env`.

Production `.app`:

```sh
make desktop-build
make desktop-run
```

Артефакт создаётся в `desktop/bin/carvionstudio.app`. Сборка подписывается ad-hoc для локального запуска; это не Developer ID signing и не notarization. Для установки/распространения потребуется отдельная release configuration.

Проверки:

```sh
make desktop-test   # root Go, Wails service Go, TypeScript, Playwright, frontend build
make lint
make build
```

Если Playwright browser ещё не установлен, выполните `cd desktop/frontend && npx playwright install chromium`. Playwright E2E работает через специальный `e2e.html` с явно синтетическим test backend; production UI module импортирует real generated Wails service binding. Native packaged smoke проверяет именно Wails app.

## Ограничения хранения и доверия

- В текущем dev режиме требуется локальная PostgreSQL и `DATABASE_URL`.
- Примените миграцию `00003_localization_engine.sql` через `make migrate-up` перед использованием Localization Engine.
- Android XML inline markup/CDATA внутри переводимых элементов пока отклоняется. JSON ограничен schema `carvion.localization/v1`.
- Языковой ZIP — CARVION localization artifact с `oem_firmware=false`; он не является OEM firmware и не устанавливается на автомобиль.
- Desktop UI не получает database connection или credentials напрямую; Go composition владеет pool.
- Hub-backed repository, remote mode, пользовательская Hub авторизация и offline policy не реализованы.
- Adapter Manager запускает только доверенные локальные процессы из `ADAPTERS_DIR`; отдельный процесс сам по себе не является OS sandbox.
- Нет сетевой загрузки адаптеров и доступа к автомобильному оборудованию.
