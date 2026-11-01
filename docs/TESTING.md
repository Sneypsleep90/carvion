# Тестирование

## Текущая стратегия

- **Unit:** Go packages тестируются без PostgreSQL через `go test ./...`.
- **SDK/RPC:** dispatch четырёх методов, неизвестный метод, invalid params, RPC error и статусы результата.
- **Plugin Manager:** discovery, manifest/API version, duplicate ID, ошибки процесса, timeout/termination и capability checks.
- **Demo adapter:** synthetic JSON, точное совпадение build+hardware, unknown vehicle/build и `strings.xml` parsing.
- **HTTP:** `httptest` для health/readiness, adapters list, request validation, structured errors и manager error mapping.
- **Core application:** fake adapter gateway/repository tests проверяют use case delegation, input validation, propagation/cancellation контекста и прямые service operations без HTTP.
- **PostgreSQL repository:** integration-tagged test проверяет реальные SELECT к `manufacturers`, `vehicle_models`, `infotainment_platforms` на применённой схеме.
- **DB connection:** существующий отдельный тест с tag `integration` проверяет connect/ping; нужен `CARVION_TEST_DATABASE_URL`, migrations он не выполняет.
- **Migrations/API smoke:** проверялись отдельно на локальной PostgreSQL при реализации Step 2.

## Фактические результаты

| Дата | Команда / действие | Результат |
|---|---|---|
| 2026-11-01 | `go test -count=1 ./...` | Пройдено: `internal/adapters/androiddemo`, `internal/config`, `internal/pluginmanager`, `internal/server`, `pkg/adaptersdk/v1`; `cmd/*` и `internal/database` не содержат обычных unit tests |
| 2026-11-01, Step 1 | `make test` | Пройдено после Core refactoring; application, bridge, composition, config, API и прежние package tests прошли |
| 2026-11-01, Step 1 | `make lint` | Пройдено: gofmt check и `go vet ./...` |
| 2026-11-01, Step 1 | `make build` | Пройдено: API и `adapter-android-demo` собраны |
| 2026-11-01, Step 1 | `make test-integration` | Пройдено на локальной PostgreSQL, migration version 2; repository прочитал три существующие catalog tables |
| 2026-11-01, Step 1 | API smoke и SIGINT shutdown | `/healthz`, `/readyz`, adapter list и synthetic compatibility вернули ожидаемые ответы; log подтвердил graceful shutdown |
| 2026-11-01, при реализации Step 2 | `make test`, `make lint`, `make build` | Пройдены |
| 2026-11-01, при реализации Step 2 | `go test -race ./internal/pluginmanager ./pkg/adaptersdk/v1` | Пройдены |
| 2026-11-01, при реализации Step 2 | Goose migration Up → Down → Up | Пройдена на локальной PostgreSQL |
| 2026-11-01, при реализации Step 2 | API smoke: health/readiness, adapters, exact demo compatibility, unknown data | Проверено вручную; ответы endpoints получены |
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

В репозитории пока нет Wails/UI tests, Desktop E2E tests, migration constraint tests или repository tests: эти компоненты не реализованы.
