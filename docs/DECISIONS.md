# Архитектурные решения

Статус отделяет решения, уже воплощённые в коде, от направления, которому ещё нужна реализация.

## D-001 — CARVION является Desktop-first продуктом

- **Статус:** реализуется; Go Core, Wails service facade и React UI находятся в репозитории.
- **Решение:** основной пользовательский продукт — Wails 3 Desktop с Go core и React/TypeScript UI. REST API не является архитектурным центром.
- **Причина:** заданная продуктовая цель CARVION.
- **Альтернатива:** продолжать API-first продукт; текущим направлением отвергнута.
- **Следствие:** UI вызывает Go Core через Wails bindings, HTTP остаётся отдельным transport для Hub/API.

## D-002 — Контракт адаптера версионируется в Go package

- **Статус:** реализовано в `pkg/adaptersdk/v1`.
- **Решение:** SDK имеет package version `v1`, manifest schema version и `api_version`.
- **Причина:** core должен проверить совместимость контракта до регистрации процесса.
- **Альтернатива:** неявно менять общий контракт; текущая реализация выбирает явную проверку версии.
- **Ограничение:** реализован и описан только API v1.

## D-003 — Адаптеры работают отдельными локальными процессами по JSON-RPC

- **Статус:** реализовано; отдельный процесс запускается на каждый вызов.
- **Решение:** newline-delimited JSON-RPC 2.0 по stdin/stdout; stdout — ответ, stderr — диагностика.
- **Причина:** изоляция lifecycle без загрузки plugin code внутрь Go процесса.
- **Альтернатива:** in-process Go plugin loading; не используется.
- **Ограничение:** отдельный процесс не является OS sandbox и имеет права пользователя CARVION.

## D-004 — Каталог адаптеров локальный и доверенный

- **Статус:** реализовано.
- **Решение:** Manager сканирует direct children `ADAPTERS_DIR`, проверяет manifest/API version и containment executable; network download отсутствует.
- **Причина:** ограничить источник исполняемого кода и оставить сбор данных под контролем ядра.
- **Альтернатива:** загружать произвольные plugins из удалённого каталога; запрещено текущими требованиями.
- **Следствие:** распространение через будущий Hub требует отдельного решения о подписи, доверии и provenance.

## D-005 — Неизвестная совместимость явно обозначается `unknown`

- **Статус:** реализовано в SDK и demo.
- **Решение:** отсутствие точного профиля не возвращает `supported`; применяются `supported`, `unsupported`, `unknown`.
- **Причина:** отсутствие доказательств нельзя представлять как проверенную поддержку.
- **Альтернативы:** считать отсутствие правил поддержкой или отрицанием; оба варианта теряют смысл неопределённости.

## D-006 — PostgreSQL остаётся основной целевой БД; Desktop deployment не решён

- **Статус:** PostgreSQL локально используется в разработке; Desktop runtime mode не решён.
- **Решение:** сохранить PostgreSQL. Текущий composition root использует локальный PostgreSQL; API и Desktop consumers не получают credentials через application service.
- **Причина:** локальная БД для разработки задана проектом; production Desktop должен работать с Hub без административных DB credentials.
- **Неизвестно:** будущий Desktop storage implementation — Hub-backed repository, локальный режим либо их конфигурация; authorization к Hub также требует решения.
- **Следующее действие:** уточнить protocol/auth и deployment при Wails/Hub integration; локальный PostgreSQL repository служит dev implementation порта.

## D-007 — Application Core отделён от транспорта и инфраструктуры

- **Статус:** реализовано в Step 1.
- **Решение:** domain и application packages не импортируют HTTP, chi, Wails, pgx или SDK. `application.Service` использует `AdapterGateway` и `CatalogRepository`; local PostgreSQL и прежний Adapter Manager подключены через infrastructure implementations.
- **Причина:** Wails должен вызывать Core напрямую, а хранилище может быть заменено на Hub-backed implementation без изменения use cases.
- **Альтернатива:** оставить orchestration внутри HTTP handlers или связать Desktop с localhost REST; не используется.
- **Следствие:** HTTP handlers остаются тонким transport и используют тот же Core; Wails service facade также вызывает Core напрямую. Remote repository пока не реализован.

## D-008 — Текущий HTTP API сохранён как отдельный Hub/API transport

- **Статус:** существующие routes сохранены; Desktop не использует их как внутренний bridge; полный Hub runtime/auth ещё не реализован.
- **Решение:** сохранить `cmd/carvion-api` и `internal/server`; они создают HTTP transport поверх Core. Desktop bindings обращаются к application service напрямую.
- **Причина:** сохранить полезный API и отделить сетевой server от Desktop use cases.
- **Альтернатива:** удалить API или заставить Desktop обращаться к локальному REST; обе отвергнуты.

## D-009 — Wails Desktop находится в отдельном Go module и вызывает Core in-process

- **Статус:** реализовано в `desktop/` на Wails v3.0.0.
- **Решение:** держать Wails host в nested module, с локальным `replace` на корневой CARVION module. React использует generated Wails TS bindings через typed `DesktopBackend`; локальный REST не применяется.
- **Причина:** Wails Desktop-specific dependencies и packaging отделены от API module, при этом бизнес-логика вызывается в том же процессе.
- **Альтернативы:** включить Wails в корневой module либо поднять HTTP bridge для UI; обе не нужны для текущего desktop scope.
- **Следствие:** Desktop и root Go packages тестируются отдельно; facade владеет lifecycle composition и показывает degraded state при ошибке локального runtime.

## D-010 — Минимальная macOS версия согласована с Go toolchain

- **Статус:** реализовано для локальной Apple Silicon сборки.
- **Решение:** объявлять minimum macOS 13.0 для Wails app и native build flags.
- **Причина:** Go 1.27 toolchain сообщил, что runtime object требует macOS 13, когда app link target был выставлен на 12.
- **Альтернатива:** сохранять macOS 12 и игнорировать несовпадение target; не выбрана.
- **Ограничение:** проверена сборка и запуск на текущей macOS 26.6.2 arm64; предыдущие версии ОС отдельно не тестировались.

## D-011 — Manifest schema v2 отделяет SDK SemVer от API version

- **Статус:** реализовано в Step 3; SDK API остаётся `v1`, manifest schema — `2`, SDK package — `1.0.0`.
- **Решение:** manifest явно содержит SDK version, capability-specific `input_types` и `resource_formats`. Объявленные входы должны согласовываться с capabilities; resource inspection обязан объявлять хотя бы один формат.
- **Причина:** Manager и Desktop должны выбирать адаптер по проверяемым возможностям и входным контрактам, а не предположениям о реализации.
- **Альтернатива:** оставить API version единственным маркером и передавать capabilities без форматов; это не позволяет заранее отсеять неподдерживаемые документы.
- **Следствие:** изменения обязательного manifest-контракта требуют bump schema; несовместимый SDK major не допускается к регистрации.

## D-012 — Runtime status отражает последний process call

- **Статус:** реализовано в Manager и Adapters screen.
- **Решение:** `available` выставляется после успешного `GetInfo` или operation call; process/RPC failure меняет его на `unavailable`. Отмена parent context сама по себе не объявляет адаптер неисправным.
- **Причина:** Desktop показывает наблюдавшееся состояние без фоновых процессов и отдельного monitoring API.
- **Альтернатива:** постоянный health status или long-lived worker; сейчас этот механизм не нужен.
- **Ограничение:** `available` не подтверждает доверие к executable, подпись или OS isolation.

## D-013 — Localization formats и package provenance заданы явно

- **Статус:** реализовано в Step 4.
- **Решение:** принимать Android `strings.xml`, фиксированную JSON schema `carvion.localization/v1` и RFC 4180 CSV; исходный документ хранится как snapshot. Языковой ZIP имеет тип `carvion-localization`, содержит validation report и SHA-256, а `oem_firmware` всегда равен `false`.
- **Причина:** поддержать ограниченный и проверяемый набор ресурсов, сохранить метаданные источника и явно отделить результат локализации от OEM firmware.
- **Альтернативы:** произвольное JSON без схемы, перепаковка оригинального архива или выдача пакета за OEM образ не обеспечивают однозначную структуру и provenance.
- **Ограничение:** XML inline markup/CDATA не поддерживаются; расширение форматов требует versioned schema или отдельного parser contract.

## D-014 — Simulator использует один локальный synthetic catalog

- **Статус:** реализовано в Step 5.
- **Решение:** `internal/simulator` является единственным источником профилей и тестового XML для Core и `adapter-android-demo`; fixture значения отмечены `CARVION TEST LAB` и не связаны с серийными системами.
- **Причина:** Core и адаптер должны проверять один и тот же тестовый профиль без копирования данных и без сетевой загрузки.
- **Альтернативы:** повторять fixtures в адаптере/тестах либо загружать profiles из сети — первое допускает drift, второе не нужно для локального сценария.

## D-015 — Core повторно подтверждает identity до localization

- **Статус:** реализовано в Step 5.
- **Решение:** Core выбирает адаптер по объявленным capabilities, требует точного совпадения `Detect` identity с выбранным synthetic профилем, затем передаёт точные platform/hardware revision/firmware build в compatibility check. Только `supported` продолжает workflow; `unknown` и `unsupported` останавливают его до импорта.
- **Причина:** не считать выбранный adapter или частичное совпадение доказательством поддержки.
- **Альтернативы:** выбирать первый установленный адаптер и доверять его единственному статусу либо продолжать при `unknown` — обе модели допускают ложный позитив.

## D-016 — Simulator run events не требуют отдельной истории

- **Статус:** реализовано в Step 5.
- **Решение:** события возвращаются только в `SimulatorRunResult` текущего Desktop вызова; успешный прогон переиспользует Localization Repository для project/translation/package persistence. Отдельная таблица прогонов не добавлена.
- **Причина:** Simulator является контролируемым тестовым сценарием, а долгосрочная операция-история не входит в текущий scope.
- **Альтернативы:** общая операция БД или только frontend state — первая требует новую persistent lifecycle-модель, вторая не сохраняет уже поддерживаемые localization artifacts.

## D-017 — Simulator сценарии ограничены по времени и отменяемы

- **Статус:** реализовано в Step 5.
- **Решение:** Core ограничивает каждый Simulator запуск двумя минутами; Wails facade связывает context с request ID, отменяет активную операцию по команде пользователя и завершает её при shutdown Desktop service.
- **Причина:** отдельный adapter process не должен бесконечно удерживать UI или оставаться активным после ухода со страницы/закрытия приложения.
- **Альтернативы:** неограниченный блокирующий вызов либо только UI timeout оставляют Core/process операции без cancellation.
- **Ограничение:** cancellation не является OS sandbox и не может принудительно ограничить произвольную системную активность сверх существующих Plugin Manager process controls.

## D-018 — Geely Research начинается с локального файлового профиля

- **Статус:** реализовано в Step 6.
- **Решение:** профиль загружается как строгий `carvion.geely-research/v1` JSON, а XML/JSON ресурс выбирается отдельным файлом. Go Core передаёт содержимое единственному `adapter-geely-research` process через существующий Plugin Manager; адаптер не читает файлы самостоятельно.
- **Причина:** доступные официальные материалы не подтверждают точные идентификаторы и совместимость конфигураций 2022–2024. Файловый импорт даёт безопасную основу для сбора подтверждённых наблюдений.
- **Альтернативы:** автоматическое подключение/сбор данных или вывод совместимости из общих спецификаций; не подходят по имеющимся доказательствам и выходят за согласованный read-only scope.

## D-019 — Geely compatibility определяется только по полному exact tuple

- **Статус:** реализовано в Step 6; начальный каталог пуст.
- **Решение:** сравниваются модель, год, рынок, комплектация, мультимедийная платформа, имя/версия ОС, аппаратный идентификатор/ревизия и firmware version/build ID. Только точная reviewed запись может вернуть `supported` или `unsupported`; incomplete/absent match остаётся `unknown`.
- **Причина:** название автомобиля и рекламные параметры не устанавливают равенство мультимедийных конфигураций между рынками/годами.
- **Альтернативы:** приблизительное совпадение модели/SoC/версии ОС — не подтверждает точную конфигурацию.

## D-020 — Research Mode передаёт результат Localization Engine по явному действию

- **Статус:** реализовано в Step 6.
- **Решение:** адаптер использует `internal/localization.Parse`; Core сверяет ключи и строки адаптера с парсером Localization Engine и вызывает существующий project import только после нажатия пользователя. Совместимость `unknown` не блокирует файловый localization workflow. Миграция не добавляется.
- **Причина:** сохранить один parser и имеющиеся правила хранения/истории; отделить анализ файла от утверждения совместимости с автомобилем.
- **Альтернативы:** отдельное Geely хранилище/парсер дублировали бы данные и поведение; автоматический импорт скрывал бы выбор пользователя.

## D-021 — SDK v1 получает additive поле versioned vehicle profile

- **Статус:** реализовано в Step 6 как SDK 1.1.0; API остаётся v1.
- **Решение:** `CompatibilityRequest.vehicle_profile` — необязательный `json.RawMessage`, а manifest может объявить `vehicle-research-profile-json`. Обычные compatibility requests сохраняют прежнюю форму.
- **Причина:** базовые identity/hardware/firmware поля не содержат year, region, trim и OS tuple, нужные exact Geely rule.
- **Альтернативы:** расширить общие доменные поля или добавить новый метод/API major — сильнее меняют существующий контракт и не нужны для одного versioned profile.

## D-022 — Production Deployment остаётся заблокированным, пока нет обязательных доказательств

- **Статус:** реализовано в Step 7; `productionDeploymentEnabled = false` в Core.
- **Решение:** Production readiness audit сохраняется как `blocked`, если нет подтверждённой платформы, exact compatibility, разрешённого install method, result verification, recovery procedure и bench evidence. Конфигурация/UI не может включить execution.
- **Причина:** в проекте нет проверенных installation/recovery данных ни для одной реальной платформы; Geely exact catalog пуст. Блокировка в Core предотвращает обход интерфейсного состояния.
- **Альтернативы:** включать Production по флагу, считать synthetic profile достаточным или продолжать при `unknown` — не дают evidence для безопасного реального действия.

## D-023 — Deployment jobs и progress events хранятся в PostgreSQL

- **Статус:** реализовано в Step 7; PostgreSQL integration execution требует отдельной test DB.
- **Решение:** сохранять job snapshot вместе с упорядоченным append-only event log. Изменение состояния и соответствующее событие пишутся одной транзакцией; UI восстанавливает состояние через Core bindings.
- **Причина:** постоянная история нужна для разбора blocked/failure/cancel состояний после перезапуска.
- **Альтернативы:** session-only log не переживает перезапуск; отдельный event broker/service неоправданны для local Desktop scope.

## D-024 — Installation capability отделена от adapter SDK

- **Статус:** реализовано в Step 7: Core provider interface не содержит `Install`, подключён только synthetic simulator.
- **Решение:** Deployment capability выражается отдельным Core provider contract; текущий provider объявляет `CanInstall=false`, а SDK JSON-RPC contract не получает install method.
- **Причина:** plugin discovery/compatibility не должны автоматически давать процессу право менять устройство.
- **Альтернативы:** добавить install RPC всем адаптерам или считать `CheckCompatibility` разрешением записи смешивает проверку данных с опасной операцией.

## D-025 — SHA-256 verification не трактуется как authenticity

- **Статус:** реализовано в package verifier и документировано в Step 7.
- **Решение:** verifier ограничивает ZIP structure и проверяет manifest/checksums/hash; UI/docs явно указывают, что digest не доказывает издателя, подпись или OEM происхождение.
- **Причина:** CARVION language package не является OEM firmware, а подписанная provenance infrastructure отсутствует.
- **Альтернативы:** называть digest проверкой подлинности создаёт неподтверждённую гарантию.
