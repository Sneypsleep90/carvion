# Тестирование

## Текущая стратегия

- **Unit:** Go packages тестируются без PostgreSQL через `go test ./...`.
- **SDK/RPC:** dispatch четырёх методов, неизвестный метод, invalid params, RPC error, SDK major mismatch, manifest schema v2 и статусы результата.
- **Plugin Manager:** trusted local discovery, manifest metadata/format errors, API/SDK mismatch, duplicate ID, process routing/exit, malformed response, RPC error, timeout/termination, parent cancellation, capability и resource format checks.
- **Demo adapter:** synthetic JSON, точное совпадение build+hardware, unknown vehicle/build и `strings.xml` parsing.
- **HTTP:** `httptest` для health/readiness, adapters list, request validation, structured errors и manager error mapping.
- **Core application:** fake adapter gateway/repository tests проверяют use case delegation, input validation, propagation/cancellation контекста и прямые service operations без HTTP.
- **Localization Engine:** unit tests проверяют XML/JSON/CSV parsing/export, metadata, duplicate keys, malformed XML/UTF-8, missing/blank translations, format placeholders, plural categories, source reimport review и воспроизводимый ZIP package.
- **Localization application:** fake repository tests проверяют import, translation status lifecycle, CSV stale-source rejection, validation persistence и package approval gate.
- **Simulator:** unit/service tests проверяют единый synthetic catalog, capability-based adapter selection, точную identity/hardware/firmware, `supported` / `unsupported` / `unknown`, early exits, RPC errors, context cancellation и package integrity. Общий двухминутный run deadline реализован, но отдельный тест его истечения не запускался; Plugin Manager RPC timeout проверяется отдельно.
- **Geely Research:** strict profile schema/UTF-8/размер/unknown fields и privacy patterns, точный compatibility matcher, empty catalog, RPC error/context, XML/JSON parser reuse, same-content handoff, path rejection и report privacy проверяются Go tests. Process test строит executable и вызывает его через Plugin Manager. PostgreSQL integration test отдельно импортирует ресурс настоящим adapter process и проверяет project/resource round-trip на выделенной базе.
- **Simulator process/DB:** integration-tagged tests компилируют настоящий demo executable, вызывают его через Plugin Manager/JSON-RPC, а отдельный сценарий проверяет сохранение Localization project/translations/package в выделенной PostgreSQL базе.
- **Localization PostgreSQL:** integration test проверяет project/resource round-trip, draft/approval history, reimport source change, validation-run persistence и package artifact storage на выделенной базе.
- **Desktop service:** Wails lifecycle tests проверяют degraded startup, direct Core delegation, readiness state, shutdown и cancelled context.
- **UI:** Playwright E2E использует отдельный synthetic test backend; production bundle импортирует generated Wails bindings. Проверяются навигация, версии/capabilities/status адаптера, `unknown` compatibility, localization import/edit/approve/validate и повторная проверка при изменении source.
- **Geely Research UI:** Playwright synthetic backend покрывает profile JSON, отдельные XML/JSON ресурсы, unverified provenance, ограниченный evidence scope, `unknown`, Localization import и отчёт без raw resource content/path. Это UI preview test, не native Wails process smoke.
- **Simulator UI:** E2E покрывает полный synthetic localization/package flow и download, unknown firmware, incompatible hardware, контролируемую RPC ошибку и user cancellation. Native Wails window automation не входит в Playwright suite.
- **Native Desktop:** `wails3 package` создаёт и подписывает ad-hoc `.app`; ручной smoke открывает приложение, читает runtime state, вызывает compatibility и `InspectResources` через настоящие Wails bindings.
- **PostgreSQL repositories:** integration-tagged tests проверяют catalog SELECT и lifecycle localization project/translation/history/validation/package на применённых migrations; localization tests также проверяют transaction rollback и запрет изменения history.
- **DB connection:** существующий отдельный тест с tag `integration` проверяет connect/ping; нужен `CARVION_TEST_DATABASE_URL`, migrations он не выполняет.
- **Migrations/API smoke:** проверялись отдельно на локальной PostgreSQL при реализации Adapter SDK (исходный Step 2).

## Фактические результаты

| Дата | Команда / действие | Результат |
|---|---|---|
| 2026-11-01 | `go test -count=1 ./...` | Пройдено: `internal/adapters/androiddemo`, `internal/config`, `internal/pluginmanager`, `internal/server`, `pkg/adaptersdk/v1`; `cmd/*` и `internal/database` не содержат обычных unit tests |
| 2026-11-01, Step 1 | `make test` | Пройдено после Core refactoring; application, bridge, composition, config, API и прежние package tests прошли |
| 2026-11-01, Step 1 | `make lint` | Пройдено: gofmt check и `go vet ./...` |
| 2026-11-01, Step 1 | `make build` | Пройдено: API и `adapter-android-demo` собраны |
| 2026-11-01, Step 1 | `make test-integration` | Пройдено на локальной PostgreSQL, migration version 2; repository прочитал три существующие catalog tables |
| 2026-11-01, Step 1 | API smoke и SIGINT shutdown | `/healthz`, `/readyz`, adapter list и synthetic compatibility вернули ожидаемые ответы; log подтвердил graceful shutdown |
| 2026-11-01, Step 2 | `make desktop-test` | Пройдено: root `go test ./...`, desktop module Go tests, `npm run typecheck`, 3 Playwright E2E tests и Vite production build. Wails Go build выдал предупреждение о macOS target 12, после чего target согласован на 13 и Wails package пересобран без предупреждения. |
| 2026-11-01, Step 2 | `make lint` | Пройдено после исключения Wails-generated `desktop/build/` platform scaffolding из gofmt scan; `go vet ./...` пройден. |
| 2026-11-01, Step 2 | `make build` | Пройдено: API и `adapter-android-demo` собраны. |
| 2026-11-01, Step 2 | `make db-up`, `make migrate-up` | PostgreSQL Compose container запущен; Goose сообщил schema version 2, миграции актуальны. |
| 2026-11-01, Step 2 | `make desktop-build` | Пройдено: Wails v3.0.0 создал `desktop/bin/carvionstudio.app`, ad-hoc codesign verification прошёл; бинарь `arm64`, bundle ID `com.carvion.studio`, minimum macOS 13.0. |
| 2026-11-01, Step 2 | `make desktop-run` + native UI smoke | Приложение запустилось. AX state показал Core ready, PostgreSQL connected и один локальный synthetic adapter. Compatibility вызов вернул exact synthetic match; `InspectResources` вернул 2 переводимые строки и пропустил `translatable=false`. Автомобиль не подключался. |
| 2026-11-01, Step 2 | npm install во время Wails packaging | npm audit сообщил 0 vulnerabilities; остаётся предупреждение npm allow-scripts для `fsevents`, package/build завершились успешно. |
| 2026-11-01, Step 3 | `make desktop-test` | Пройдено: root Go suite, nested Desktop Go suite, TypeScript typecheck, 4 Playwright E2E tests и production Vite build. В браузерных E2E Wails предупреждает, что доступен только UI preview; test backend synthetic. |
| 2026-11-01, Step 3 | `go test -race ./internal/pluginmanager ./pkg/adaptersdk/v1` | Пройдено: Plugin Manager и SDK/RPC без race reports. Включает process launch/routing, SDK mismatch, bad manifest/result, timeout, RPC error и cancellation status. |
| 2026-11-01, Step 3 | `make lint` | Пройдено после gofmt для `internal/core/domain/adapter.go` и `internal/core/domain/errors.go`; `go vet ./...` завершился успешно. Первый запуск указал на эти два файла. |
| 2026-11-01, Step 3 | `make build` | Пройдено: собраны API binary и `adapter-android-demo`. |
| 2026-11-01, Step 3 | `make desktop-build` | Пройдено: Wails CLI v3.0.0 пересоздал bindings, собрал arm64 macOS `.app` и проверил ad-hoc codesign. npm audit — 0 vulnerabilities; выдано предупреждение `allow-scripts` для `fsevents`. |
| 2026-11-01, Step 3 | Native app smoke | Упакованный CARVION Studio открылся с Core ready, PostgreSQL connected и зарегистрированным `adapter-android-demo`; Adapters screen показал manifest metadata/status. Через UI и реальные Wails bindings compatibility вернула `supported` для точного synthetic profile; XML inspection вернула две переводимые строки и пропустила `translatable="false"`. Smoke выполнялся вручную, не в CI; автомобиль не подключался. |
| 2026-11-01, Step 0 | Integration-tagged `TestConnect` | Не запускался в аудите: `CARVION_TEST_DATABASE_URL` тогда не был задан; позже при Step 1 integration suite прошёл на локальной PostgreSQL |
| 2026-11-01, Step 4 | `go test ./internal/localization ./internal/core/... ./internal/composition` | Пройдено: parser, validator, deterministic package builder, application use cases, Core и composition tests |
| 2026-11-01, Step 4 | Migration `00003_localization_engine.sql` | Goose успешно применил миграции до schema version 3 на отдельной тестовой PostgreSQL базе |
| 2026-11-01, Step 4 | `go test -tags=integration -count=1 ./internal/core/infrastructure/postgres` | Пройдено на выделенной базе после миграций: localization persistence, rollback при duplicate resource, immutable history, source-change review и package artifact storage |
| 2026-11-01, Step 4 | `make test-integration` | Пройден полный integration-tagged Go suite с `CARVION_TEST_DATABASE_URL`, включая PostgreSQL repository tests |
| 2026-11-01, Step 4 | `go test -tags=integration -count=1 ./...` | После review fix повторно пройден полный integration suite на мигрированной выделенной базе без test cache |
| 2026-11-01, Step 4 | `make desktop-test` | Пройдено: root и Desktop Go tests, TypeScript typecheck, 5 Playwright E2E tests и Vite production build |
| 2026-11-01, Step 4 | `go test -race ./internal/localization ./internal/core/application` | Пройдено без race reports |
| 2026-11-01, Step 4 | `make lint` и `make build` | Пройдено: gofmt/vet, API и `adapter-android-demo` binaries собраны |
| 2026-11-01, Step 4 | `make desktop-build` | Пройдено: Wails v3 собрал production frontend и macOS `.app`, ad-hoc codesign verification прошёл. Native UI smoke для этого этапа отдельно не запускался. npm сообщил allow-scripts warning для `fsevents`; сборка успешна. |
| 2026-11-01, Step 5 | `make desktop-test` | Пройдено после финальной cancellation event правки: root/Desktop Go suite, TypeScript, 10 Playwright E2E tests и production Vite build. UI E2E использует synthetic backend, Wails browser-preview warning ожидаем. |
| 2026-11-01, Step 5 | `go test -tags=integration -count=1 ./...` | Пройдено после Goose migrations 1–3 на изолированной временной PostgreSQL базе. Включены JSON-RPC subprocess full flow и Localization persistence; база удалена после теста. |
| 2026-11-01, Step 5 | `go test -race -count=1 ./internal/simulator ./internal/adapters/androiddemo ./internal/core/application` и Desktop `go test -race -count=1 ./...` | Пройдено без race reports. Для Desktop native build задан minimum macOS 13.0 через CGO/macOS environment flags. |
| 2026-11-01, Step 5 | `make lint`, `make build`, `make desktop-build` | Пройдены: gofmt/vet, API/demo executables, Wails frontend и macOS app package с успешной ad-hoc codesign verification. npm сообщил `allow-scripts` warning для `fsevents`, build успешен. |
| 2026-11-01, Step 5 | `git diff --check` | Пройдено после удаления одной лишней пустой строки в конце изменённого Localization frontend file. |
| 2026-11-01, Step 6 | `make desktop-test` | Пройдено: все root и nested Desktop Go tests, TypeScript typecheck, 13 Playwright E2E tests и Vite production build. Geely сценарии проверили ранний отказ для oversized files, XML import/report/privacy и отдельный versioned JSON resource. Wails browser-preview warning ожидаем; E2E использует test backend. |
| 2026-11-01, Step 6 | `make lint` | Пройдено: gofmt check и `go vet ./...`. |
| 2026-11-01, Step 6 | `make build` | Пройдено: API binary, `adapter-android-demo` и `adapter-geely-research` собраны. |
| 2026-11-01, Step 6 | `make desktop-build` | Пройдено: Wails CLI v3.0.0 пересоздал bindings и собрал/подписал ad-hoc macOS `.app`; codesign verification прошёл. npm audit — 0 vulnerabilities; остаётся предупреждение allow-scripts для `fsevents`. |
| 2026-11-01, Step 6 | `go test -tags=integration ./internal/core/infrastructure/postgres -run '^$' -count=1` | Пройдено только компилирование integration package/test. Geely PostgreSQL test фактически не запускался: `CARVION_TEST_DATABASE_URL` не задан. |
| 2026-11-01, Step 6 | `make desktop-run` / native UI smoke | Не запускались: в worktree отсутствуют `.env` и `DATABASE_URL`; `docker compose ps` сообщил, что compose DB credentials не заданы. Wails packaging и Wails facade unit tests выполнены. |
| 2026-11-02, Step 7 | `make desktop-test` | Пройдено: root Go suite, Desktop Go suite, TypeScript strict check, 18 Playwright E2E tests и production Vite build. E2E использует synthetic backend; Wails browser-preview предупреждение ожидаемо. |
| 2026-11-02, Step 7 | `make test-integration` | Пройдено на отдельном временном PostgreSQL 18 container с migrations 00001–00004; полный integration-tagged Go suite, включая deployment jobs/events, прошёл. Container удалён после проверки, dev DB не использовалась. |
| 2026-11-02, Step 7 | `make lint`, `make build` | Пройдены: gofmt/vet, API binary и оба локальных adapter executables собраны. |
| 2026-11-02, Step 7 | `make desktop-build` | Пройдено: Wails CLI v3.0.0 собрал macOS arm64 `.app`; ad-hoc signing verification прошёл. npm audit — 0 vulnerabilities; остаётся предупреждение allow-scripts для optional `fsevents`. |
| 2026-11-02, Step 7 | Geely subprocess test под полной нагрузкой | Первый integration run выявил test-only 3s startup timeout на `GetInfo`; он увеличен до 10s, production RPC timeout не менялся. Полный integration suite после изменения прошёл. |
| 2026-11-02, Step 7 | Physical hardware / native UI smoke | Не выполнялись. Synthetic E2E или package verification не подтверждают работу на автомобиле или стенде. |

## Команды разработчика

```sh
make test
make lint
make build
```

Для отдельной DB connection проверки используйте выделенную тестовую базу:

```sh
CARVION_TEST_DATABASE_URL='postgres://user:password@localhost:5432/carvion_test?sslmode=disable' go test -tags=integration ./...
```

Wails UI E2E использует synthetic backend и не заменяет native manual smoke. Полноценной автоматизации управления Wails native window в CI пока нет. PostgreSQL integration tests требуют отдельную заранее созданную и мигрированную test database; migrations запускаются отдельно от Go test suite.
