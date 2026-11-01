# Известные проблемы и технический долг

Проблемы ниже найдены статическим аудитом. Они не означают, что соответствующая функциональность уже реализована. Серьёзность не ранжирована без deployment requirements и threat model.

## Архитектура и продукт

- **Отдельные экраны пока не готовы:** Language Packages и Operation History — placeholders, а Settings ограничены runtime details. Package build доступен из Localization.
- **API покрывает не все SDK operations:** HTTP предоставляет список и compatibility check; `Detect` и `InspectResources` доступны через Core/Manager, но routes отсутствуют.
- **Repository coverage частичная:** `CatalogRepository` читает только `manufacturers`, `vehicle_models`, `infotainment_platforms`; `LocalizationRepository` поддерживает localization data. Нет repositories/use cases для adapter definitions, versions, hardware variants, firmware builds и compatibility rules.
- **Нет связи DB catalog и manifest discovery:** Manager регистрирует адаптеры в памяти; PostgreSQL adapter tables не синхронизируются с локальными manifests.
- **Hub-backed storage отсутствует:** application repository port позволяет подключить remote implementation, но клиент Hub, API endpoints для каталога и auth ещё не реализованы.
- **Desktop PostgreSQL deployment не решён:** локальная БД используется при разработке; production клиентская схема (Hub, offline policy, migrations) и auth требуют решения. Desktop facade не отдаёт DB credentials frontend.
- **Android XML parsing boundary:** nested inline markup, CDATA, comments, processing instructions и namespaced roots/`string`/`item` resources отклоняются; требуется format-preserving strategy до их поддержки.
- **Locale validation:** проверяется непустое значение, но синтаксис BCP 47 не валидируется.
- **macOS distribution не готов:** локальная `.app` собирается для arm64 и проходит ad-hoc signature check; Developer ID signing, notarization и обновление приложения не настроены.

## Config, dependencies, operations

- **Config deployment mode:** runtime config разделён на Core `Config` и `HTTPConfig`; Compose по-прежнему использует `POSTGRES_*`, а local Core подключается через `DATABASE_URL`. Desktop remote profile ещё не описан.
- **Repository history начинается с текущего backend:** до публикации workspace не содержал `.git`; более ранние commits и branches отсутствуют в локальной истории.
- **Объём monorepo не подтверждён:** в workspace доступен Go backend; наличие или состояние соседних частей `carvion` нельзя проверить.
- **HTTP exposure:** default `HTTP_HOST` — `127.0.0.1`, но его можно настроить на внешний interface; auth, mTLS и rate limiting отсутствуют. Не публиковать API наружу без отдельного решения и защиты.
- **Process isolation ограничено:** timeout, bounded frames и environment allowlist не образуют OS sandbox; процесс наследует права пользователя.
- **Generated binaries:** `bin/carvion-api` и demo executable создаются `make build`, игнорируются Git и не являются source.
- **npm install warning:** Wails packaging сообщает, что install scripts `fsevents` ещё не одобрены в npm `allow-scripts`; текущий `npm install` и сборка успешны, audit нашёл 0 vulnerabilities.
- **Desktop CI coverage:** текущий GitHub workflow не запускает nested `desktop/` tests, TypeScript/Playwright или macOS package build; эти команды проверялись вручную на Apple Silicon.
- **Повторяющиеся metadata/status invariants:** demo manifest повторяет значения `GetInfo()` в Go, а Manager сверяет их при регистрации; правки надо синхронизировать. Проверка допустимого compatibility status независимо реализована в SDK, Manager и demo adapter. Это небольшой maintenance debt, а не подтверждённый функциональный дефект.

## Данные и безопасность

- **Реальных автомобильных данных нет:** demo profiles synthetic; Geely или другой производитель не поддерживается.
- **Нет firmware deployment:** установка, изменение и запись firmware отсутствуют.
- **Simulator ограничен тестовыми fixtures:** profiles, строки и переводы демонстрационные; event log хранится только в текущем результате, а настоящий Wails native UI не имеет автоматизированного UI-теста.
- **Simulator workflow создаёт localization data:** локальные тестовые project/package записи сохраняются в существующих localization tables; отдельной истории прогонов нет.
- **Simulator run deadline не имеет отдельного expiry test:** общий двухминутный предел задан в Core; протестированы cancellation и Plugin Manager RPC timeout, но не ожидание истечения полного run deadline.
- **Нет network plugin source:** будущая доставка из Hub требует отдельной модели доверия.

## Тестовое покрытие

- Go unit/service tests, `make lint`, `make build`, Desktop TypeScript/E2E/build и native macOS smoke прошли 2026-11-01; фактические команды записаны в [TESTING.md](TESTING.md) и [simulator/TEST_RESULTS.md](simulator/TEST_RESULTS.md).
- PostgreSQL integration test проверяет чтение Foundation tables, rollback импортной транзакции и immutable translation history; все schema constraints ещё не покрыты отдельными automated tests.
- Wails lifecycle имеет unit tests, UI имеет Playwright E2E с fake backend; полноценного native-window automation в CI нет. JSON-RPC process boundary и PostgreSQL persistence Simulator покрыты отдельными integration tests.
