# CARVION

CARVION — Desktop-платформа для анализа и локализации ресурсов автомобильных мультимедийных систем. CARVION Studio использует Go Core и подключаемые адаптеры; основной интерфейс — desktop-приложение.

## Возможности

- Localization Engine для Android XML и JSON-ресурсов, с проверкой переводов, CSV import/export и сборкой CARVION language package.
- Adapter SDK и Plugin Manager для запуска адаптеров отдельными процессами с JSON-RPC 2.0.
- Simulator для тестирования workflow на встроенных демонстрационных профилях.
- Go HTTP API предоставляет backend endpoints; PostgreSQL хранит catalog и localization data.

Симулятор работает только с тестовыми данными. Подключения к автомобилям, реальные адаптеры и установка прошивок не реализованы.

## Технологии

- **Core:** Go, domain/application/infrastructure layers.
- **Desktop:** Wails 3, React, TypeScript, Tailwind CSS, shadcn/ui-компоненты, Lucide.
- **Хранилище:** PostgreSQL, SQL migrations через Goose.
- **Адаптеры:** версионированный Go SDK, JSON-RPC 2.0 через stdin/stdout.
- **Проверки:** Go tests, Playwright.

Документация проекта: [docs/README.md](docs/README.md).

Исходный код: [github.com/Sneypsleep90/carvion](https://github.com/Sneypsleep90/carvion).
