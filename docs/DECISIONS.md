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
