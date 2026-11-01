# Ближайшие задачи

Step 7 реализовал только безопасный тестовый Deployment Engine. Production Mode остаётся заблокированным; Geely compatibility catalog пуст. Общий порядок этапов: [ROADMAP.md](ROADMAP.md).

## Следующий этап — Step 8: CARVION Hub & Commercial Platform

1. Спроектировать Hub-backed repositories и авторизацию Desktop; не передавать клиенту PostgreSQL credentials.
2. Определить доверенную provenance/signing модель для распространяемых adapter packages и обновлений.
3. Согласовать Hub API и синхронизацию локальных adapter manifests с PostgreSQL catalog.
4. До реализации сетевой доставки кода подготовить threat model, политику обновлений и revoke/rollback для приложений/плагинов.

Production deployment — отдельный нерешённый вопрос. Любое будущее рассмотрение требует проверенных технических данных и разрешённых процедур для конкретной конфигурации; текущий шаг не подтверждает ни одну платформу.

## Доступно сейчас

- CARVION Studio напрямую вызывает Go Core через Wails bindings.
- Adapters показывает локальные процессы; `adapter-android-demo` и `adapter-geely-research` работают через JSON-RPC.
- Simulator проходит synthetic сценарии без настоящего автомобиля.
- Localization импортирует Android XML и versioned JSON, редактирует/проверяет русский перевод и строит CARVION language package.
- Geely Research Mode принимает JSON-профиль и отдельный XML/JSON ресурс, показывает публичные источники с scope, классифицирует без точного правила как `unknown`, импортирует ресурс по явному действию и экспортирует отчёт без local path/raw resource content.
- Deployment workspace создаёт persistent jobs, проверяет CARVION package и может выполнить только синтетическую симуляцию. Production readiness audit записывает blockers; Core запрещает Production Mode.

## Оставшиеся ограничения

- Geely hardware revisions и firmware build IDs для целевых лет не подтверждены; exact catalog пуст.
- PostgreSQL integration test нового Geely import требует отдельный `CARVION_TEST_DATABASE_URL` и в текущем запуске не выполнялся.
- Desktop dev режим требует локальную PostgreSQL; Hub repository и Hub auth не реализованы.
- Android XML nested inline markup/CDATA пока отклоняется.
- Language Packages и Operation History остаются отдельными placeholder экранами.
- OS sandbox, plugin signing, native UI automation, macOS notarization и distribution update ещё не реализованы.
- Deployment PostgreSQL integration tests требуют отдельной test database; production installation/recovery и испытания на стенде отсутствуют.
