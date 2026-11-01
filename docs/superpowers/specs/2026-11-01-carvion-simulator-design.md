# CARVION Simulator Design

**Статус:** архитектурный вариант одобрен пользователем 2026-11-01; требуется review этого документа перед реализацией.

## Цель

Добавить CARVION Studio Simulator Mode для прогона synthetic мультимедийных профилей через существующие Core, Adapter SDK/Plugin Manager и Localization Engine. Сценарии не подключаются к автомобилю и не имитируют установку firmware.

## Подтверждённый текущий контекст

- `internal/core/application.AdapterGateway` — transport-neutral порт к адаптерам.
- `internal/core/infrastructure/adapterbridge` вызывает адаптеры через Plugin Manager и JSON-RPC subprocess.
- `adapter-android-demo` предоставляет SDK `Detect`, `CheckCompatibility`, `InspectResources`; его профили встроены как test fixtures.
- Localization use cases импортируют Android XML, сохраняют переводы, валидируют их и собирают ZIP package.
- `desktop.ApplicationService` — Wails boundary к Core; Desktop не вызывает HTTP API.
- `Operation History` пока не сохраняет общую историю; Simulator будет показывать историю текущей сессии.

## Архитектура

1. Создать `internal/simulator` с единственным каталогом synthetic profiles, scenario/result/event types и тестовым XML. Core и trusted local `adapter-android-demo` используют общий каталог; копии profile data не расходятся. Не добавлять SDK version или новые HTTP routes.
2. Добавить Core `SimulatorService`. Он перечисляет synthetic profiles, фильтрует discovered adapters по `Detect`, `CheckCompatibility`, `InspectResources` и формату XML, затем маршрутизирует операции через имеющийся `AdapterGateway`.
3. Для выбранного профиля Simulator получает synthetic identity, hardware revision, exact firmware build и test XML fixture. `Detect`, compatibility check и resource inspection проходят тем же Plugin Manager/JSON-RPC путём, который использует Core для адаптеров.
4. На совместимом профиле Core запускает существующие Localization use cases: импорт демонстрационных китайских строк, запись фиксированных synthetic русских переводов, утверждение, validation и package build. Переводные fixture-значения явно обозначаются как демонстрационные.
5. Перед успешным итогом Core проверяет SHA-256 artifact, читаемость ZIP, ожидаемый resource file и manifest с `package_type=carvion-localization` и `oem_firmware=false`.
6. `desktop.ApplicationService` публикует типизированные Wails методы для списка профилей и запуска сценария. Simulator screen показывает badge `SIMULATION`, выбранный профиль, adapter, этапы, события, результат совместимости и package verification.

Simulator run log входит в результат сценария и хранится в состоянии текущей Desktop-сессии. Новая PostgreSQL таблица и глобальный persistent `Operation History` не входят в этап.

## Synthetic profiles and outcomes

- **Demo Android Platform A / B:** известные synthetic platform, hardware revision, firmware build и локальные XML строки; ожидаемый результат — `supported`, затем полный localization cycle.
- **Unknown Firmware:** известные synthetic identity и hardware, не зарегистрированный build; ожидаемый результат — `unknown`, цикл останавливается до импорта и package build.
- **Incompatible Hardware:** известная synthetic identity/build с несовместимой аппаратной ревизией; ожидаемый результат — `unsupported`, цикл останавливается до импорта.
- **Simulated communication failure:** controlled profile сценарий вызывает JSON-RPC error; test timeout/cancellation покрывает потерю RPC responsiveness. Core возвращает структурированную ошибку и завершает run без успешного package result.

Все названия и payloads содержат явный synthetic/test marker. Они не называют и не моделируют поддержку конкретной реальной марки автомобиля.

## Desktop workflow

Пользователь открывает Simulator, выбирает профиль и запускает сценарий. UI по мере выполнения показывает event log и состояния `running`, `supported`, `unsupported`, `unknown`, `failed` или `completed`. Для supported профиля результат содержит ссылку на созданный localization project и проверенный CARVION package, который можно скачать. Для остальных профилей показывает причину отказа. Никакой UI action не сообщает об установке/записи в автомобиль.

## Ошибки и ограничения

- У каждого run и RPC шага есть context deadline; отмена UI останавливает дальнейшие шаги. RPC error, timeout и cancellation отражаются как failure events.
- Plugin Manager/adapter errors не превращаются в `supported`; результат run получает status `failed` с безопасным сообщением.
- Localization project/package остаётся помеченным synthetic; пакет не является OEM firmware.
- Текущая PostgreSQL composition нужна для сохранения project/translations/package; Simulator не получает DB pool и не передаёт его адаптеру.
- Profile list и run log текущего этапа локальны и не синхронизируются через Hub.

## Проверка

- Unit tests: profile validation, capability-based adapter selection, supported/unknown/unsupported routing, deterministic event order, RPC error/timeout/cancellation paths и package integrity verifier.
- Integration tests: запуск `adapter-android-demo` как отдельного JSON-RPC процесса, полный успешный сценарий через Core и PostgreSQL, отказ перед package creation для unknown/incompatible profiles.
- Wails service tests: direct delegation/context propagation.
- Playwright E2E: запуск supported full cycle и отображение событий/ZIP verification; unknown firmware, incompatible hardware и RPC failure не создают успешный package result.

Все проверки используют локальные synthetic fixtures. Native vehicle connection и firmware installation не тестируются.

## Вне scope

- Подключение к реальному автомобилю, CAN/OBD или OEM hardware.
- Firmware installation, flashing, modification или bypass защиты.
- HTTP routes для Simulator.
- Постоянный общий operation history, новая БД-схема, Hub mode или remote simulator profiles.
- Автоматический перевод; русские строки — фиксированные synthetic QA fixtures.
