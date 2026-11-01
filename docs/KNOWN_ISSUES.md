# Известные проблемы и технический долг

Проблемы ниже найдены статическим аудитом. Они не означают, что соответствующая функциональность уже реализована. Серьёзность не ранжирована без deployment requirements и threat model.

## Архитектура и продукт

- **Desktop отсутствует:** нет Wails, UI и bindings. Core application service готов для прямого вызова из Go, но Desktop host ещё не создан.
- **API покрывает не все SDK operations:** HTTP предоставляет список и compatibility check; `Detect` и `InspectResources` доступны через Core/Manager, но routes отсутствуют.
- **Repository пока read-only и неполный:** `CatalogRepository` читает только `manufacturers`, `vehicle_models`, `infotainment_platforms`. Нет repositories/use cases для adapter definitions, versions, hardware variants, firmware builds и compatibility rules.
- **Нет связи DB catalog и manifest discovery:** Manager регистрирует адаптеры в памяти; PostgreSQL adapter tables не синхронизируются с локальными manifests.
- **Hub-backed storage отсутствует:** application repository port позволяет подключить remote implementation, но клиент Hub, API endpoints для каталога и auth ещё не реализованы.
- **Desktop PostgreSQL deployment не решён:** локальная БД используется при разработке. Для будущего Desktop/Hub flow ещё нужны авторизация, offline policy, миграции и способ управления local dev DB; административные DB credentials не входят в Core service.

## Config, dependencies, operations

- **Config deployment mode:** runtime config разделён на Core `Config` и `HTTPConfig`; Compose по-прежнему использует `POSTGRES_*`, а local Core подключается через `DATABASE_URL`. Desktop remote profile ещё не описан.
- **Repository history начинается с текущего backend:** до публикации workspace не содержал `.git`; более ранние commits и branches отсутствуют в локальной истории.
- **Объём monorepo не подтверждён:** в workspace доступен Go backend; наличие или состояние соседних частей `carvion` нельзя проверить.
- **HTTP exposure:** default `HTTP_HOST` — `127.0.0.1`, но его можно настроить на внешний interface; auth, mTLS и rate limiting отсутствуют. Не публиковать API наружу без отдельного решения и защиты.
- **Process isolation ограничено:** timeout, bounded frames и environment allowlist не образуют OS sandbox; процесс наследует права пользователя.
- **Generated binaries:** `bin/carvion-api` и demo executable создаются `make build`, игнорируются Git и не являются source.
- **Повторяющиеся metadata/status invariants:** demo manifest повторяет значения `GetInfo()` в Go, а Manager сверяет их при регистрации; правки надо синхронизировать. Проверка допустимого compatibility status независимо реализована в SDK, Manager и demo adapter. Это небольшой maintenance debt, а не подтверждённый функциональный дефект.

## Данные и безопасность

- **Реальных автомобильных данных нет:** demo profiles synthetic; Geely или другой производитель не поддерживается.
- **Нет firmware deployment:** установка, изменение и запись firmware отсутствуют.
- **Нет network plugin source:** будущая доставка из Hub требует отдельной модели доверия.

## Тестовое покрытие

- Unit/service tests, `make test`, lint, build и PostgreSQL integration test прошли 2026-11-01; фактические команды записаны в [TESTING.md](TESTING.md).
- PostgreSQL integration test проверяет чтение Foundation tables, но SQL schema constraints не покрыты отдельными automated tests.
- UI, Wails lifecycle, packaging и Desktop E2E не имеют кода или тестов.
