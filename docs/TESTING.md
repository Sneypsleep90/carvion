# Тестирование

## Текущая стратегия

- **Unit:** Go packages тестируются без PostgreSQL через `go test ./...`.
- **SDK/RPC:** dispatch четырёх методов, неизвестный метод, invalid params, RPC error, SDK major mismatch, manifest schema v2 и статусы результата.
- **Plugin Manager:** trusted local discovery, manifest metadata/format errors, API/SDK mismatch, duplicate ID, process routing/exit, malformed response, RPC error, timeout/termination, parent cancellation, capability и resource format checks.
- **Demo adapter:** synthetic JSON, точное совпадение build+hardware, unknown vehicle/build и `strings.xml` parsing.
- **HTTP:** `httptest` для health/readiness, adapters list, request validation, structured errors и manager error mapping.
- **Core application:** fake adapter gateway/repository tests проверяют use case delegation, input validation, propagation/cancellation контекста и прямые service operations без HTTP.
- **Desktop service:** Wails lifecycle tests проверяют degraded startup, direct Core delegation, readiness state, shutdown и cancelled context.
- **UI:** Playwright E2E использует отдельный synthetic test backend; production bundle импортирует generated Wails bindings. Проверяются навигация, версии/capabilities/status адаптера, фильтр по capability/формату, выбор adapter ID, `unknown` compatibility и Android XML inspection.
- **Native Desktop:** `wails3 package` создаёт и подписывает ad-hoc `.app`; ручной smoke открывает приложение, читает runtime state, вызывает compatibility и `InspectResources` через настоящие Wails bindings.
- **PostgreSQL repository:** integration-tagged test проверяет реальные SELECT к `manufacturers`, `vehicle_models`, `infotainment_platforms` на применённой схеме.
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

Wails UI E2E использует synthetic backend и не заменяет native manual smoke. Полноценной автоматизации управления Wails native window в CI пока нет. Migration constraint tests и automated PostgreSQL schema tests отсутствуют; repository integration покрывает только исходные catalog tables.
