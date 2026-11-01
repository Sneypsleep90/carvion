# CARVION Geely Research Adapter — Implementation Plan

> Выполнять после одобрения пользователем спецификации `docs/superpowers/specs/2026-11-01-geely-research-adapter-design.md`. Использовать TDD для изменений кода: сначала тест, подтвердить ожидаемый fail, затем минимальная реализация.

**Цель:** добавить локальный Geely Research Adapter, который импортирует versioned JSON-наблюдение и отдельный XML/JSON resource document, возвращает `unknown` без точной проверенной записи, передаёт ресурсы в существующий Localization Engine и экспортирует очищенный диагностический отчёт.

**Архитектура:** `internal/geelyresearch` содержит схему профиля, проверенные публичные evidence entries, строгую валидацию и exact matcher. `adapter-geely-research` реализует существующий SDK через trusted JSON-RPC subprocess. Core оркестрирует адаптер и Localization Engine. Wails напрямую предоставляет типизированные методы экрану Desktop; HTTP routes и новые миграции не добавляются.

**Технологии:** Go, существующий Adapter SDK v1 с additive minor extension, Plugin Manager/JSON-RPC, существующий `internal/localization.Parse`, pgx LocalizationRepository, Wails 3, React/TypeScript, Playwright.

## Task 1: Profile schema, provenance, evidence catalog

**Файлы:** создать `internal/geelyresearch/profile.go`, `evidence.go`, `compatibility.go`, `report.go`, соответствующие `*_test.go` и пустой `compatibility_records.json` по versioned schema.

- [x] Написать тесты на строгий `carvion.geely-research/v1` decode, допустимые observation fields, unknown/missing значения, отклонение VIN/секретов/лишних полей и размер профиля.
- [x] Написать тесты, что source reference в user JSON не получает статус verified; официальный evidence entry фиксирует publisher, URL, publication date, retrieval date, scope и confidence.
- [x] Написать table-driven exact matcher тесты по каждому полю: модель/год/рынок/trim/platform/OS/hardware/build; missing field и отсутствие записи дают `unknown`; только explicit exact record может вернуть `supported` или `unsupported`.
- [x] Запустить `go test ./internal/geelyresearch -count=1`; первое выполнение показало ожидаемую ошибку компиляции отсутствующих типов и функций.
- [x] Реализовать профиль, strict validation, компактную curated evidence catalog для найденных официальных публикаций и пустой каталог compatibility rules. Публичные сведения 2021/2025/2026 явно ограничить их источником и периодом.
- [x] Повторить `go test ./internal/geelyresearch -count=1` — пройдено.

## Task 2: Additive SDK support for full vehicle profile

**Файлы:** изменить `pkg/adaptersdk/v1/types.go`, `types_test.go`, `rpc_test.go`, при необходимости SDK protocol docs.

- [x] Добавить падающие тесты на необязательное поле `vehicle_profile` в `CompatibilityRequest`, отсутствие поля в старых/обычных payloads и input type для Geely research profile.
- [x] Проверить текущий strict RPC decoding: обычные запросы без расширения остаются валидны; Geely adapter использует SDK 1.1.x / API v1 и manifest declares required inputs.
- [x] Реализовать additive поле и input type, сохранить API major `v1`, обновить SDK minor version и manifest validation, не менять существующие методы и ответы.
- [x] Убедиться, что JSON-RPC framed request с максимальным профилем остаётся в `MaxRPCMessageBytes`; зафиксировать предел профиля в `internal/geelyresearch`.
- [x] Запустить `go test ./pkg/adaptersdk/v1 -count=1` и `go test ./internal/pluginmanager ./internal/core/infrastructure/adapterbridge -count=1` — пройдены; существующее сравнение SDK request обновлено для `json.RawMessage`.

## Task 3: Geely research adapter process

**Файлы:** создать `internal/adapters/geelyresearch/adapter.go`, тесты, `cmd/adapter-geely-research/main.go`, `adapters/adapter-geely-research/manifest.json`; изменить `Makefile` build targets.

- [x] Добавить тесты `GetInfo`/manifest; `Detect` для file observation с непроверенной идентичностью; compatibility unknown при пустом каталоге; exact matcher delegation; malformed profile, bad SDK input and cancellation.
- [x] Добавить тесты `InspectResources` на Android `strings.xml`, string-array/plurals и `carvion.localization/v1` JSON, используя shared `internal/localization.Parse`; некорректный XML/JSON, UTF-8 и лимит размера должны возвращать ошибку.
- [x] Запустить `go test ./internal/adapters/geelyresearch -count=1` до реализации — ожидаемое отсутствие command path; после реализации тест прошёл.
- [x] Реализовать SDK adapter без сетевого I/O, filesystem reads, device API, PostgreSQL и firmware actions. `Detect` всегда обозначает импортированное file observation, а не vehicle connection.
- [x] Создать manifest/executable layout; добавить сборку обоих adapter processes в `make build` и Desktop dev/build prerequisities.
- [x] Проверить реальный subprocess/RPC invocation через Plugin Manager, manifest registration, форматы ресурсов и RPC error. Размеры профиля/ресурса проверены на соответствие RPC frame; timeout/cancellation проверяются существующими Plugin Manager tests и adapter cancellation tests.
- [x] Запустить `go test ./internal/geelyresearch ./internal/adapters/geelyresearch -count=1`, SDK/Plugin Manager/bridge tests и `make build-adapter` — пройдены.

## Task 4: Core research and Localization use cases

**Файлы:** создать `internal/core/domain/geely_research.go`, `internal/core/application/geely_research.go` и тесты; изменить application/domain compatibility contracts, adapter bridge и `internal/composition/local.go` только при необходимости.

- [x] Написать application tests на профиль validation, Geely adapter ID/capabilities, status propagation, `unknown` при пустом каталоге, контекст/ошибки и report sanitization.
- [x] Написать tests на анализ ресурса через adapter и последующий explicit import того же содержимого через существующий `LocalizationService`; неизвестная совместимость не блокирует файловую локализацию.
- [x] Написать тест, что source path не передаётся в gateway/RPC, а report исключает path и raw XML content; profile decoder отклоняет VIN-like и credential assignments.
- [x] Запустить `go test ./internal/core/application ./internal/core/infrastructure/adapterbridge -count=1`; ожидаемые ошибки отсутствующих методов/types были исправлены, повторный запуск пройден.
- [x] Реализовать Core workflows и передачу optional `vehicle_profile` по AdapterGateway/bridge. Profile/research results остаются session-scoped; Localization project сохраняется существующим repository.
- [x] Добавить PostgreSQL integration case к `LocalizationRepository` tests для persistence выбранного resource file; новая миграция не добавлялась.
- [x] Повторить Core unit tests; integration package компилируется. Полный DB execution остался не выполнен: `CARVION_TEST_DATABASE_URL` в среде отсутствует.

## Task 5: Typed Wails service boundary

**Файлы:** изменить `desktop/application_service.go`, `desktop/application_service_test.go` и сгенерированные `desktop/frontend/bindings/`.

- [x] Добавить падающие facade tests для evidence/profile/resource/localization/report calls, Wails context и недоступного Core.
- [x] Реализовать typed Wails methods над Core use cases, сохраняя lifecycle и прямой вызов Core.
- [x] Regenerate Wails bindings установленной версией Wails v3.0.0; статусы и JSON report сериализуются как typed values/string artifact.
- [x] Запустить Desktop Go suite через `make desktop-test`.

## Task 6: Geely Research Mode UI and E2E

**Файлы:** изменить `desktop/frontend/src/navigation.ts`, `App.tsx`, `lib/backend.ts`, `e2e-entry.tsx`, Playwright tests; создать `features/geely-research-page.tsx`.

- [x] Добавить Playwright tests для JSON profile, XML/JSON resources, field provenance, public evidence scope, `unknown`, Localization import и privacy report download.
- [x] Создать экран «Исследование Geely» с двумя file inputs, error/loading/empty states, evidence dates/scopes и маркировкой `из файла / не проверено`.
- [x] Выводить compatibility status из adapter exact catalog; текущий empty catalog даёт `unknown`, без заявления соединения.
- [x] Реализовать XML/JSON analysis и explicit Localization project import через Wails, сохранив файл/формат в Engine.
- [x] Реализовать JSON report download без path/raw resource content.
- [x] Отклонять превышающие 128 KiB profile/resource files до чтения в Desktop; Go catalog validator не принимает общие product/naming sources как доказательство точной совместимости.
- [x] Подключить generated bindings в production UI; E2E backend остаётся synthetic и помеченным как fixture.
- [x] Запустить `make desktop-test`: TypeScript, 13 Playwright E2E и production frontend build прошли.

## Task 7: Documentation and final verification

**Файлы:** создать `docs/research/geely/OVERVIEW.md`, `PLATFORMS.md`, `FIRMWARE.md`, `SECURITY.md`, `OPEN_QUESTIONS.md`, `EVIDENCE.md`; обновить `docs/research/README.md`, `docs/README.md`, `docs/architecture.md`, `docs/PROJECT.md`, `docs/ROADMAP.md`, `docs/PROGRESS.md`, `docs/CHANGELOG.md`, `docs/DECISIONS.md`, `docs/TESTING.md`, `docs/KNOWN_ISSUES.md`, `docs/NEXT_STEPS.md`, `docs/DESKTOP.md`, `docs/adapters/SDK.md`, `PLUGIN_PROTOCOL.md`, `SECURITY.md` и `CREATING_ADAPTER.md` по фактическим изменениям.

- [x] Для каждого публичного Geely факта указать URL, издателя, дату публикации при наличии, retrieval date, confidence и scope; unknown по 2022–2024 задокументирован.
- [x] Документировать profile schema, evidence/provenance, RPC fields, SDK minor change, пустую compatibility table, limits, report privacy и XML parser limits.
- [x] Обновить roadmap/progress после фактических изменений и результатов.
- [x] Выполнить `make desktop-test`, `make lint`, `make build`, `make desktop-build` и `git diff --check`; PostgreSQL integration test добавлен/компилируется, но требует отсутствующий в среде `CARVION_TEST_DATABASE_URL`.
- [x] Проверить сборку приложения; native run не выполнялся, потому что отсутствуют `.env`, `DATABASE_URL` и Compose DB credentials.
- [x] Сверить docs с manifest, SDK types, Core and UI; проверенные Geely build IDs и support claims не добавлялись.

## Завершение

Шаг завершён, когда Geely manifest успешно регистрируется локальным Plugin Manager, profile/resource workflow вызывается через Wails напрямую, Localization Engine получает выбранный ресурс, diagnostic report экспортируется без приватных полей, а Geely совместимость остаётся `unknown` без exact verified record. Все выполненные тесты перечислены с командами и результатами.
