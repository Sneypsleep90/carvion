# CARVION Step 7 — Deployment Engine Design

**Статус:** направление одобрено пользователем 2026-11-02. Production deployment remains blocked until every readiness requirement below has verified evidence.

## Аудит готовности

| Условие | Состояние в репозитории | Следствие |
|---|---|---|
| Реально поддерживаемая платформа | Нет. Android профили явно synthetic; Geely exact compatibility catalog пуст | Нет target, пригодного для production deployment |
| Подтверждённый способ установки | Не описан и не реализован; текущий Adapter SDK поддерживает только Detect, CheckCompatibility и InspectResources | Production installation capability отсутствует |
| Точное правило совместимости | Synthetic exact rules есть только в demo; реальных записей нет | Synthetic совпадение разрешено только в test provider |
| Подтверждённое восстановление | Recovery procedure/capability не найдены | Production preflight блокируется |
| Стендовые испытания | В репозитории есть software tests, но нет hardware bench records | Production preflight блокируется |

Вывод: не создавать, не запускать и не предлагать реальную запись. Целевой результат Step 7 — Deployment Engine с симулятором и постоянным аудитом, при этом production deny-by-default в Go Core.

## Цель и границы

Пользователь выбирает существующий Localization project и целевой профиль. Core строит/получает его CARVION language package, независимо проверяет ZIP integrity, запускает preflight и записывает job/events в локальный PostgreSQL. Доступен только synthetic provider. UI и Core различают итог `simulated` от подтверждённой установки.

Production preflight возвращает `blocked` с перечнем отсутствующих доказательств. Проверка выполняется в Core до обращения к любому deployment provider. Нет production provider, vehicle IO, установки, подписи OEM-прошивок, обхода Secure Boot/авторизации или привилегированных команд. HTTP routes не добавляются.

## Рассмотренные варианты

1. **Отдельный Deployment Engine и provider contract — выбран.** Он держит installation/recovery capability отдельно от inspection Adapter SDK, сохраняет PostgreSQL job/event audit и добавляет Wails progress events. Сейчас зарегистрирован только synthetic provider.
2. **Добавить `InstallPackage` в существующий Adapter SDK.** Отклонён: открывает реальную mutation surface до подтверждения платформы, установки, восстановления и bench испытаний; текущий SDK/RPC frame также не рассчитан на передачу deployment artifacts.
3. **Только frontend mock.** Отклонён: бизнес-правила безопасности должны исполняться в Core, а задания и audit должны переживать перезапуск.

## Компоненты и границы

```text
React Deployment screen
    ├── Wails typed calls ──> deployment application service / Core
    └── Wails `deployment:progress` events <── Core event publisher
                                 │
                                 ├── LocalizationService → approved project/package
                                 ├── PackageVerifier → integrity-only verification
                                 ├── DeploymentProvider port → synthetic provider only
                                 └── DeploymentRepository → PostgreSQL jobs + append-only events
```

- `internal/core/domain`: DeploymentJob, mode, state, preflight check, event and result types.
- `internal/core/application`: job orchestration, safety policy, package preflight, strict state transitions and provider selection.
- `internal/core/infrastructure/postgres`: job repository; update current job and append its event atomically.
- `internal/deployment`: bounded CARVION ZIP verifier and synthetic-only provider implementation, following existing module boundaries.
- `desktop/`: Wails methods for list, preflight/start, get job and cancel job; Wails application events stream progress. No frontend transport or business rules.
- `desktop/frontend`: Deployment page selects a Localization project and synthetic profile, starts a simulated job, shows checks, progress, persisted history and explicit production blockers.

### Provider capability contract

Deployment has a provider contract separate from the existing Adapter SDK inspection capabilities. Installation, result verification and recovery-check capabilities are distinct declarations. A provider's synthetic execution capability is not production install capability. No production provider or callable production installation method is registered in this step. A future provider requires a separate reviewed design and explicit validated capability/evidence before use.

## Production readiness policy

Production remains disabled by a Core-owned fail-closed policy. UI state or adapter response cannot enable it. Every future production job must have all of the following before a provider can be considered:

1. An exact, reviewed compatibility record for the complete target tuple and package format.
2. A documented, authorized installation method and a provider explicitly declaring installation capability.
3. An independently declared result-verification capability and documented verification criteria.
4. A documented recovery method and provider recovery capability for the same target.
5. Recorded hardware bench test evidence and an accepted safety review.
6. A CARVION package passing structural, manifest, validation and checksum checks.

Any missing, stale, conflicting or `unknown` condition blocks production. Current blockers include all items 1–5. Production preflight can create a persisted blocked job for review, but cannot call a provider or start an installation state.

## Package integrity

- Deployment consumes a package produced by the existing Core Localization Engine for the selected project; the UI does not send arbitrary paths or install images.
- Verify the stored artifact SHA-256, ZIP CRC/readability, bounded compressed/uncompressed size, exact allowed entry layout, no duplicate/traversal paths, strict `manifest.json`, expected `carvion.language-package/v1` schema and `package_type=carvion-localization`, `oem_firmware=false`, successful validation, and every `checksums.sha256` entry.
- The verifier reports `integrity verified`, not signature/authenticity, OEM approval or vehicle compatibility. CARVION packages remain localization artifacts and are never reinterpreted as firmware.
- Corrupt, oversized, malformed, path-traversing, checksum-mismatched or invalid packages stop before provider execution.

## Preflight and state machine

Preflight checks package integrity, selected profile provenance, exact compatibility outcome, provider mode/capabilities, result-verification capability, recovery-check capability and production evidence. Production uses no live device discovery; in this step no real profile can pass. Test mode accepts only profiles marked synthetic from the embedded simulator catalog.

States are explicit and persisted with these transitions: `queued → preflighting`; from `preflighting`, either `blocked` (terminal) or `preflight-passed → simulating → verifying → simulated` (terminal). Provider/integrity failures become `failed` (terminal); cancellation from a running state becomes `cancelled` (terminal). `simulated` is terminal and must never serialize/display as `installed`, `deployed` or production success. State transitions are validated in Core. Progress events include job ID, monotonic event sequence, state/stage and a user-readable sanitized message. The event is committed before best-effort live Wails emission; clients can reload the persisted job/events if the UI missed a live event.

On simulated failure, the engine checks the provider's recovery simulation capability and records the result. It does not claim a rollback occurred. Production recovery is always unavailable in this step.

## PostgreSQL audit

- Add a migration for `deployment_jobs` and `deployment_job_events`. The job stores project ID, package SHA-256, target profile ID when applicable, mode, provider ID, current state/result, sanitized reason and timestamps. Events reference job ID with ordered sequence, timestamp, state, stage and sanitized message.
- Events are append-only through repository API and database trigger; a job-state update and its event insert occur in one transaction.
- Persist accepted, blocked, failed, cancelled and simulated jobs. Keep only required identifiers and hashes; do not store package bytes, local filesystem paths, raw imported profiles, VINs, credentials or firmware content in deployment history.
- Local PostgreSQL owner access is not tamper-proof audit security. Document that a local administrator can alter the database; remote signed audit is out of scope.

## Desktop workflow

- Add a Deployment nav item following current CARVION Studio design.
- Show the production status as `blocked`, with the concrete missing conditions above. The user can request a production preflight record; there is no production install action.
- Test mode selects a Localization project/package and an embedded synthetic profile, then starts only the synthetic provider. Show live progress, a cancel action for the active test, and persistent run history.
- Present package integrity separately from compatibility/installation. Final wording says `Только симуляция; автомобиль не изменён`.
- Show errors with failed stage and next safe action. Recovery UI states `проверка возможности восстановления`; simulation is not real recovery.

## Verification plan

- Unit tests: readiness fail-closed for each missing condition; unknown/unsupported target stops; synthetic profile may run only in test mode; strict state transition table; package valid/corrupt/path traversal/duplicate entry/ZIP bomb/checksum mismatch/oem marker/invalid validation; provider error, timeout, cancellation and result verification; simulated recovery wording.
- PostgreSQL integration tests: job/event transactional ordering, append-only event constraints, stable ordering/history retrieval and persistence of blocked/simulated/failed outcomes.
- Wails service tests: Core-only delegation, lifecycle context, progress event envelope, cancellation and no production provider call.
- Playwright E2E: production blockers visible and auditable, valid test package runs synthetic flow, events and final `simulated` label, failure/cancel state, persisted history rendering. Frontend fixtures remain synthetic.
- Run Go tests, Desktop tests, lint, API/adapters build, Wails arm64 package and PostgreSQL integration only when a dedicated test database is configured. No physical hardware tests are possible in this step; report that explicitly.

## Documentation

Create `docs/deployment/ARCHITECTURE.md`, `PREFLIGHT.md`, `SAFETY.md`, `RECOVERY.md`, and `TESTING.md`. Update the README index, PROJECT, architecture, ROADMAP, PROGRESS, CHANGELOG, DECISIONS, TESTING, KNOWN_ISSUES and NEXT_STEPS. Every hardware claim must remain `not tested` unless an actual bench record is added with provenance. The current Step 7 result is simulator-only.
