# Протокол Plugin Manager

## Жизненный цикл

1. При старте Go Core сканирует прямые дочерние каталоги `ADAPTERS_DIR` (по умолчанию `adapters`).
2. Из каждого каталога читает один `manifest.json`, проверяет контракт и безопасное расположение executable.
3. Запускает процесс с `GetInfo`. Адаптер регистрируется только если ответ точно совпадает с manifest.
4. Для каждой операции запускается отдельный процесс, ему передаётся ровно один запрос; после одного ответа процесс завершается.

Плагин выбирается ядром по adapter ID. Desktop выбирает ID из списка capability-совместимых адаптеров; транспорт не запускается из React.

## JSON-RPC 2.0

Сообщения — по одному JSON object на строку, UTF-8:

```json
{"jsonrpc":"2.0","id":"1","method":"GetInfo","params":{}}
```

Успех (manifest v2, текущая версия SDK):

```json
{"jsonrpc":"2.0","id":"1","result":{"manifest_version":2,"api_version":"v1","sdk_version":"1.1.0","id":"adapter-example","name":"Example Adapter","version":"1.0.0","executable":"bin/adapter-example","capabilities":{"detect":true,"check_compatibility":true,"inspect_resources":false},"input_types":["vehicle-observation-json","compatibility-profile"],"resource_formats":[]}}
```

Ошибка использует стандартный `error` объект JSON-RPC с `code` и `message`. Неизвестные поля, неверный JSON-RPC version/request ID, отсутствующий `result`, лишние frames или некорректная форма результата считаются ошибкой процесса.

Поддерживаемые методы: `GetInfo`, `Detect`, `CheckCompatibility`, `InspectResources`. Request/result shapes определены JSON tags в `pkg/adaptersdk/v1/types.go` и `rpc.go`.

## SDK 1.1 compatibility extension

`CheckCompatibility` может дополнительно получить `vehicle_profile` как произвольный JSON object, если поле объявлено Core сценария и адаптер описывает тип входа `vehicle-research-profile-json`. Для Geely Research это строгая схема `carvion.geely-research/v1`; обычные compatibility requests не содержат это поле. Оно передаётся в том же ограниченном RPC frame и не открывает адаптеру самостоятельный доступ к файлам, сети или устройствам.

## I/O и лимиты

- stdin: один request frame; Core закрывает stdin после отправки.
- stdout: только один response frame; diagnostics отправляйте в stderr.
- RPC request/response frame: максимум 1 MiB.
- manifest: максимум 64 KiB.
- default deadline: 3 секунды на вызов.
- максимум: 4 одновременных процесса на adapter ID.
- окружение: allowlist `PATH`, `LANG`, `LC_ALL`, `TMPDIR`.

Ошибка запуска, ненулевой exit, timeout, malformed frame/result и RPC error приводят к `unavailable`. Отмена родительского context возвращается вызывающему коду; она сама по себе не доказывает неисправность адаптера. Успешный вызов возвращает состояние `available`.

## Данные

Ядро передаёт adapter input как содержимое RPC request. Оно не передаёт PostgreSQL connection, credential, файловый путь к источнику или device handle. Сбор данных из внешних источников должен оставаться отдельной, контролируемой ядром возможностью с собственной моделью разрешений.
