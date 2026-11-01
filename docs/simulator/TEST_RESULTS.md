# CARVION Simulator — результаты проверок

## Что покрыто

- `internal/simulator`: стабильный порядок profiles, synthetic marker, обязательные поля, UTF-8 fixture и defensive copy.
- `internal/adapters/androiddemo`: Platform A/B detection, exact firmware/hardware compatibility, unknown identity/build, XML parsing и controlled RPC error.
- Core use cases: capability selection, проверка identity после Detect, ordered events, ранние exits, context cancellation, ошибка RPC, отказ при invalid package hash/ZIP/resources/checksums/non-OEM manifest. Общий двухминутный run deadline задан в коде; отдельный тест именно его истечения не выполнялся. Plugin Manager RPC timeout проверяется существующими тестами.
- Process integration: настоящий `adapter-android-demo` компилируется как отдельный executable, обнаруживается Plugin Manager через manifest, вызывается через JSON-RPC stdin/stdout и завершает полный сценарий с тестовым localization repository.
- PostgreSQL integration: отдельный тестовый adapter process проходит тот же Core/RPC сценарий, project/translations/package сохраняются через Localization Repository в тестовой БД.
- Wails service: прямое делегирование Core use cases, lifecycle context, request-scoped cancellation и shutdown.
- Playwright E2E: полный supported flow/download, unknown firmware, incompatible hardware, controlled RPC error и отмена. UI test backend synthetic и не подменяет реальные Core/RPC/DB integration tests.

## Фактически выполнено в Step 5

| Проверка | Результат |
|---|---|
| `make desktop-test` | Пройдено: root и Desktop Go suite, `npm run typecheck`, все 10 Playwright E2E и production Vite build. E2E выводит ожидаемое Wails browser-preview предупреждение. |
| `go test -tags=integration -count=1 ./...` | Пройдено на отдельной временной PostgreSQL базе после Goose migrations до version 3; включены JSON-RPC process scenario и PostgreSQL localization persistence. Временная база удалена после теста. |
| `go test -tags=integration ./internal/core/application -run TestSimulatorScenarioRunsThroughDemoJSONRPCProcess -count=1 -v` | Пройдено ранее отдельным запуском: реальный subprocess, Plugin Manager и JSON-RPC со stub localization repository. |
| `go test -race -count=1 ./internal/simulator ./internal/adapters/androiddemo ./internal/core/application` | Пройдено без race reports. |
| `cd desktop && MACOSX_DEPLOYMENT_TARGET=13.0 CGO_CFLAGS='-mmacosx-version-min=13.0' CGO_LDFLAGS='-mmacosx-version-min=13.0' go test -race -count=1 ./...` | Пройдено без race reports. |
| `make lint` | Пройдено: gofmt check и `go vet ./...`. |
| `make build` | Пройдено: API и demo adapter executables собраны. |
| `make desktop-build` | Пройдено: Wails v3 production frontend и macOS `.app` собраны, ad-hoc codesign verification прошёл. npm сообщил `allow-scripts` warning для `fsevents`; сборка завершилась успешно, audit сообщил 0 vulnerabilities. |
| `git diff --check` | Пройдено после удаления лишней конечной пустой строки из ранее изменённого Localization UI. |

Integration suite создавался на отдельной базе вида `carvion_step5_<timestamp>` с локальным `DATABASE_URL`; перед запуском применялись Goose migrations `00001`–`00003`, после завершения временная база удалялась. В рабочую development database тестовые сценарии не записывали.

## Ограничения результата

- Playwright проверяет экран через synthetic `DesktopBackend`; это не native Wails window automation.
- `make desktop-build` собрал `.app`, но отдельный native UI smoke именно экрана Simulator не запускался.
- PostgreSQL integration требуется запускать только на выделенной тестовой базе с применёнными migrations.
- Отдельный тест, который ждёт истечения двухминутного Simulator deadline, не запускался; RPC-level timeout покрыт тестами Plugin Manager.
- Профили, strings, firmware/build identifiers и русские переводы — тестовые; реального автомобиля или прошивки нет.
