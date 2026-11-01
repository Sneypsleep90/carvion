# Система адаптеров

## Состояние

**Реализовано и проверено:** `pkg/adaptersdk/v1`, process JSON-RPC, `internal/pluginmanager`, manifest discovery/validation и synthetic `adapter-android-demo`. В новом roadmap это Step 3; интеграция с Desktop core ещё впереди.

**Не реализовано:** Hub delivery, OS sandbox, remote plugins, real-vehicle data acquisition, реальные platform profiles и firmware deployment.

## Контракт SDK

SDK v1 определяет `AdapterManifest`, `VehicleIdentity`, `FirmwareInfo`, `HardwareInfo`, `CompatibilityResult`, `AdapterCapabilities` и операции:

- `GetInfo` — вернуть manifest адаптера.
- `Detect` — обработать наблюдение, переданное ядром.
- `CheckCompatibility` — вернуть `supported`, `unsupported` или `unknown`.
- `InspectResources` — разобрать переданное ядром содержимое ресурса.

Текущий API version — `v1`, manifest schema version — `1`. Подробности wire protocol см. в [adapter-sdk.md](../adapter-sdk.md).

## Регистрация и вызов

`ADAPTERS_DIR` по умолчанию `adapters`. Manager рассматривает прямые дочерние каталоги; в каждом ожидаются `manifest.json` и указанный исполняемый файл. Проверяются JSON fields, manifest/API version, ID, semantic version, capabilities, executable containment и совпадение `GetInfo` с manifest. Некорректные entries логируются и пропускаются.

Для каждого вызова Manager запускает отдельный процесс и отправляет один JSON-RPC 2.0 frame через stdin; stdout содержит ответ, stderr — диагностику. Default timeout — 3 секунды, frame limit — 1 MiB, параллельность — до четырёх вызовов на adapter. Процесс получает минимальное окружение, без PostgreSQL URL, connection pool, device handles и путей к исходным файлам.

Отдельный процесс не является OS sandbox. В каталоге должны находиться только файлы, которым доверяет владелец машины.

## Синтетический пример

`adapter-android-demo` распознаёт JSON тестовой фикстуры, допускает compatibility только при совпадении synthetic platform, hardware revision и firmware build/version, извлекает переводимые строки из переданного `strings.xml`. Неизвестные автомобиль и сборка дают `unknown`. Это не автомобильная база, не Geely adapter и не тест реального оборудования.

## Следующая интеграция

Application port и infrastructure bridge к Manager уже реализованы в Step 1. В Step 2 нужно подключить application service к Wails binding facade напрямую. Не переносить HTTP handlers в Desktop и не связывать UI с process RPC напрямую. Проверять SDK operations из Core и передавать ошибки через app-level contracts.
