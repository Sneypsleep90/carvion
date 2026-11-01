# Документация CARVION

Этот каталог описывает фактическое состояние репозитория и целевой Desktop-first план. Сначала читайте документы из таблицы. Новая проектная документация написана на русском. Подробная англоязычная заметка SDK сохранена; прежний архитектурный снимок вынесен в архив, потому что на текущей файловой системе `architecture.md` и `ARCHITECTURE.md` не могут быть отдельными файлами.

## Навигация

| Документ | Содержание |
|---|---|
| [PROJECT.md](PROJECT.md) | Назначение CARVION, цели и текущее состояние возможностей |
| [ARCHITECTURE.md](architecture.md) | Текущая и целевая архитектура, HTTP-зависимости и границы компонентов |
| [TECH_STACK.md](TECH_STACK.md) | Установленные зависимости и планируемый Desktop-стек |
| [ROADMAP.md](ROADMAP.md) | Этапы Step 0–8 в заданном порядке |
| [PROGRESS.md](PROGRESS.md) | Журнал по датам с компонентами, проверками и оставшимися задачами |
| [CHANGELOG.md](CHANGELOG.md) | История технических изменений |
| [DECISIONS.md](DECISIONS.md) | Архитектурные решения, причины и альтернативы |
| [TESTING.md](TESTING.md) | Стратегия проверок и фактические результаты |
| [KNOWN_ISSUES.md](KNOWN_ISSUES.md) | Архитектурные пробелы, ограничения и технический долг |
| [NEXT_STEPS.md](NEXT_STEPS.md) | Ближайшие задачи Step 7 — Deployment Engine scope и safety design |
| [DESKTOP.md](DESKTOP.md) | Интерфейс CARVION Studio, Wails bindings и локальные команды |
| [adapters/README.md](adapters/README.md) | Индекс SDK и Plugin Manager |
| [adapters/SDK.md](adapters/SDK.md) | Версии manifest и контракты SDK |
| [adapters/PLUGIN_PROTOCOL.md](adapters/PLUGIN_PROTOCOL.md) | JSON-RPC и lifecycle process adapter |
| [adapters/CREATING_ADAPTER.md](adapters/CREATING_ADAPTER.md) | Руководство по созданию адаптера |
| [adapters/SECURITY.md](adapters/SECURITY.md) | Границы доверия и контроля ядра |
| [localization/ENGINE.md](localization/ENGINE.md) | Компоненты и жизненный цикл локализации |
| [localization/FORMATS.md](localization/FORMATS.md) | Android XML, JSON schema и CSV |
| [localization/VALIDATION.md](localization/VALIDATION.md) | Проверки, статусы и устаревшие утверждения |
| [localization/PACKAGES.md](localization/PACKAGES.md) | Состав, SHA-256 и provenance language packages |
| [simulator/OVERVIEW.md](simulator/OVERVIEW.md) | Границы Simulator, Core flow, статусы и package verification |
| [simulator/SCENARIOS.md](simulator/SCENARIOS.md) | Synthetic profiles и ожидаемые результаты сценариев |
| [simulator/TEST_RESULTS.md](simulator/TEST_RESULTS.md) | Фактические проверки Simulator и их ограничения |
| [deployment/ARCHITECTURE.md](deployment/ARCHITECTURE.md) | Core orchestration, provider boundary, state machine и persistence |
| [deployment/PREFLIGHT.md](deployment/PREFLIGHT.md) | Production readiness аудит и проверки Test/Production modes |
| [deployment/SAFETY.md](deployment/SAFETY.md) | Fail-closed гарантии и границы обещаний |
| [deployment/RECOVERY.md](deployment/RECOVERY.md) | Отказы, cancellation и отсутствие production recovery |
| [deployment/TESTING.md](deployment/TESTING.md) | Покрытие Deployment и фактические результаты |
| [research/README.md](research/README.md) | Индекс технических исследований платформ |
| [research/geely/OVERVIEW.md](research/geely/OVERVIEW.md) | Реализованный файловый Research Adapter и его ограничения |
| [research/geely/PLATFORMS.md](research/geely/PLATFORMS.md) | Факты и неизвестные по платформам |
| [research/geely/FIRMWARE.md](research/geely/FIRMWARE.md) | Firmware/build IDs и точные правила сравнения |
| [research/geely/SECURITY.md](research/geely/SECURITY.md) | Границы локальных файлов и отчёта |
| [research/geely/OPEN_QUESTIONS.md](research/geely/OPEN_QUESTIONS.md) | Неизвестные технические параметры |
| [research/geely/EVIDENCE.md](research/geely/EVIDENCE.md) | Источники, даты и область подтверждения |

## Статусные обозначения

- **Реализовано и проверено** — функция есть в коде и имеет успешную проверку.
- **Реализовано, но не проверено** — код есть, но подтверждённой проверкой не покрыт.
- **Запланировано** — работа отсутствует и относится к будущему этапу.
- **Неизвестно / требует исследования** — в репозитории недостаточно оснований для вывода.

## Сохранённые заметки предыдущих этапов

- [`adapter-sdk.md`](adapter-sdk.md) сохранён как историческая запись исходного SDK этапа; актуальный контракт и протокол описаны в `adapters/`.
- [DESKTOP.md](DESKTOP.md) описывает Wails-приложение, Localization workflow, ограничения и запуск на macOS.

Исторический архитектурный снимок исходного API/SDK этапа: [`history/architecture-foundation-adapter-sdk.md`](history/architecture-foundation-adapter-sdk.md).

Начальная точка проекта: [README.md](../README.md). Инструкции для будущих сессий Codex: [AGENTS.md](../AGENTS.md).

Краткие команды запуска Desktop и API: [RUN.md](../RUN.md).
