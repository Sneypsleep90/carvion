# CARVION Simulator

## Назначение

Simulator прогоняет контролируемые сценарии CARVION без автомобиля. Он демонстрирует обнаружение профиля, выбор локального адаптера, точную проверку compatibility, чтение синтетического Android XML, подготовку русских переводов и проверку CARVION language package.

Все профили и ресурсы созданы для тестов. Идентификатор производителя в фикстурах — `CARVION TEST LAB`. Сведения не взяты из серийных автомобилей или прошивок; поддержку Geely и других производителей этот модуль не заявляет.

## Компоненты

- `internal/simulator` хранит единственный встроенный каталог профилей и XML fixture. Неизвестный ID вне этого каталога отклоняется.
- `internal/adapters/androiddemo` использует общий каталог; отдельный процесс сохраняет контракт Adapter SDK v1 и JSON-RPC 2.0.
- `internal/core/application/simulator.go` выбирает доступный адаптер по объявленным `Detect`, `CheckCompatibility`, `InspectResources` и `android-strings-xml`, затем управляет сценарием.
- `desktop/ApplicationService` публикует `ListSimulatorProfiles` и `RunSimulatorScenario` через Wails bindings, а `CancelSimulatorScenario` отменяет запущенный вызов по request ID. Core ограничивает каждый прогон двумя минутами. Desktop не использует HTTP для вызова Core.

Операции адаптера проходят через `AdapterGateway` и Plugin Manager. У адаптера нет прямого доступа к PostgreSQL или оборудованию. Обнаружение профилей не скачивает код и данные из сети.

## Успешный сценарий

1. Core принимает только локально объявленный `profile_id`.
2. Capability discovery находит доступный адаптер и вызывает его `Detect` через JSON-RPC.
3. После успешного обнаружения Core передаёт точные identity, hardware revision, firmware build и version в `CheckCompatibility`.
4. Только результат `supported` позволяет адаптеру прочитать embedded `resources-zh-CN.xml`.
5. Core импортирует те же байты в Localization Engine и сверяет список/порядок строк, возвращённый адаптером и Core parser.
6. Для фиксированных тестовых ключей создаются и утверждаются русские переводы, выполняется validation и строится CARVION ZIP.
7. Перед успешным результатом Core проверяет размер и SHA-256 ZIP, читаемость ZIP, `manifest.json`, non-OEM marker, validation report, ресурс и все записи `checksums.sha256`.

Run events существуют только в результате текущего Desktop вызова. Localization project, переводы, validation report и пакет сохраняются существующей Localization Repository; отдельная таблица Simulator history не добавлена.

## Статусы и ошибки

- `supported`: обнаружен синтетический профиль и точные тестовые значения прошли адаптерную проверку.
- `unsupported`: известная тестовая сборка не соответствует аппаратной ревизии; localization/package шаги не выполняются.
- `unknown`: сборка не известна адаптеру; localization/package шаги не выполняются.
- `failed`: ошибка RPC, истёкший timeout, отмена контекста, ошибка импорта/валидации или проверки пакета. Причина и последняя операция помещаются в структурированный event log.

Некорректный вход и profile ID вне списка возвращают Go error до запуска сценария. Ошибки операций во время сценария возвращаются как `SimulatorRunResult` со статусом `failed`, чтобы UI мог показать события.

## Границы

Simulator только читает встроенный XML и готовит демонстрационный localization package. Он не подключается к автомобилю, не читает CAN/OBD/ADB, не устанавливает пакет и не изменяет прошивку. ZIP отмечен `package_type=carvion-localization` и `oem_firmware=false`.

## Проверки

Фактически выполненные команды и различие между Go process integration и Playwright test backend описаны в [TEST_RESULTS.md](TEST_RESULTS.md). Варианты профилей и ожидаемые отказы — в [SCENARIOS.md](SCENARIOS.md).
