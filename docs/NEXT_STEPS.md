# Ближайшие задачи: Step 2 — Wails Desktop Foundation

Step 1 Desktop-First Core Refactoring завершён. Следующая цель — добавить Desktop shell и подключить Go UI facade к `application.Service` напрямую. Полный roadmap: [ROADMAP.md](ROADMAP.md).

## План работ

1. **Создать Wails 3 entry point.** Desktop startup использует отдельную composition функцию, вызывает Go Core напрямую и не стартует loopback REST API как посредник для UI.
2. **Добавить Go binding facade.** Фасад держит lifecycle context от Wails, вызывает `application.Service` и преобразует domain results в структуры, пригодные для binding. Он не должен зависеть от HTTP handlers.
3. **Собрать минимальный React/TypeScript shell.** Добавить выбранные проектом Tailwind CSS и shadcn/ui только в нужном объёме; не добавлять localization workflow до Step 4.
4. **Подключить read-only базовые экраны.** Использовать Core для списка адаптеров и доступного каталога. Проверку compatibility включать как тонкий экран над существующим Core use case.
5. **Сохранить локальный dev storage.** Для разработки Wails может использовать `composition.NewLocal` и локальный PostgreSQL. Не добавлять административные DB credentials в UI или в Hub connection settings.
6. **Сохранить HTTP API отдельно.** `cmd/carvion-api` остаётся отдельным Hub/API transport; Desktop не вызывает `/api/v1/...` на localhost.
7. **Проверить lifecycle и ошибки.** Проверить startup failure, cancellation/shutdown, UI bindings и что context передаётся в Core. Сохранить API tests и process manager tests.
8. **Синхронизировать документацию.** После этапа обновить архитектуру, roadmap/progress, changelog, decisions, testing, known issues и next steps.

## Открытые вопросы для Step 2

- Какая минимальная версия macOS/Windows/Linux входит в desktop MVP и какие Wails build targets требуются?
- Какой профиль запускается в Desktop MVP: local PostgreSQL development mode или уже Hub-backed remote mode? Hub client/auth пока не реализованы, поэтому не заявлять удалённый сценарий как рабочий.
- Как Go facade оборачивает методы Core, принимающие `context.Context`, с учётом Wails binding API?

## Критерии завершения

- Desktop shell запускается как Wails app на выбранных целевых ОС.
- UI вызывает Core через Go bindings, без локального REST hop.
- Доменная логика остаётся в Core, а UI/HTTP — transport/presentation layers.
- Запуск и закрытие приложения освобождают Core resources; ошибки и отмена context обработаны.
- `make test`, `make lint`, `make build` и релевантная Desktop verification реально выполнены и записаны в `TESTING.md`/`PROGRESS.md`.
