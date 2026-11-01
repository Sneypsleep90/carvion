# Deployment Engine — проверки

## Автоматические проверки

Пакет verifier, Core orchestration и synthetic provider покрыты Go unit/service tests. Они проверяют валидный/повреждённый ZIP, hash/checksum/schema, production blockers, неизвестный/unsupported profile, provider capabilities, failure, timeout, cancellation, persisted event order и synthetic result/recovery checks.

PostgreSQL integration tests для `deployment_jobs` и `deployment_job_events` покрывают транзакционные transitions, sequence, immutable events, blocked Production job и чтение через новое подключение. Тестовая БД должна быть выделенной и мигрированной.

Playwright E2E использует только E2E backend в браузерном UI preview. Он проверяет blocked Production, synthetic run, package integrity failure до simulation, provider RPC failure, cancellation и восстановление выбранной задачи из истории.

## Фактические результаты текущей сессии

- `make desktop-test` — пройдено: root Go tests, Wails Go tests, TypeScript, 18 Playwright E2E tests и Vite production build.
- `make lint` — пройдено: gofmt и `go vet ./...`.
- `make build` — пройдено: API, `adapter-android-demo` и `adapter-geely-research`.
- `make desktop-build` — пройдено на macOS arm64; Wails v3.0.0 собрал `.app` и выполнил ad-hoc codesign verification.
- Полный `make test-integration` — пройдено на временном отдельном PostgreSQL 18 container после Goose migrations 00001–00004; deployment jobs/events и остальные integration tests прошли. Container автоматически удалён; пользовательская/dev БД не использовалась.
- `git diff --check` — пройдено.

Один полный integration run до исправления test-only RPC timeout упал, потому что Geely process test использовал трёхсекундный startup timeout под общей нагрузкой. Timeout самого test runner увеличен до 10 секунд; production Plugin Manager timeout не менялся. Полный integration suite после исправления прошёл.

Никаких physical hardware или bench tests не выполнялось. Native Wails UI smoke в этой сессии не запускался.
