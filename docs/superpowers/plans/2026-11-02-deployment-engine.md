# CARVION Step 7 — Deployment Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Добавить Deployment Engine с постоянным аудитом и тестовым provider, который никогда не выдаёт симуляцию за реальную установку; заблокировать production deployment в Core.

**Architecture:** `internal/core/application` управляет job, preflight и допустимыми переходами; PostgreSQL хранит jobs и append-only events. Структурный verifier проверяет только CARVION language packages, Wails передаёт Core progress и события, а единственный подключённый provider исполняет синтетический сценарий. Production provider/call path в этом шаге отсутствует.

**Tech Stack:** Go, PostgreSQL/pgx без ORM, Goose SQL migration, Wails v3 typed bindings/events, React/TypeScript, Playwright.

---

## Изменяемые файлы и границы

| Путь | Ответственность |
|---|---|
| `internal/core/domain/deployment.go` | Job, режимы, состояния, preflight checks, events и результат |
| `internal/deployment/package_verifier.go` | Чистая проверка CARVION ZIP integrity/structure, без установки или filesystem доступа |
| `internal/deployment/simulated_provider.go` | Synthetic-only provider; progress, result verification и recovery simulation |
| `internal/core/application/deployment.go` | Preflight/job lifecycle, production blocker policy, выбор provider и audit event sequencing |
| `internal/core/infrastructure/postgres/deployment_repository.go` | Persistent jobs и atomic state/event transitions |
| `db/migrations/00004_deployment_engine.sql` | Таблицы и constraints для Deployment audit |
| `internal/composition/local.go` | Wiring PostgreSQL repository и только synthetic provider в Core |
| `desktop/application_service.go`, `desktop/main.go` | Wails facade, lifecycle/cancel и `deployment:progress` event |
| `desktop/frontend/src/features/deployment-page.tsx` | Экран выбора проекта/профиля, production blockers и test simulation |
| `desktop/frontend/src/{App.tsx,navigation.ts,lib/backend.ts}` | Навигация и типизированный Wails API/events |
| `desktop/frontend/e2e-entry.tsx`, `desktop/frontend/e2e/studio.spec.ts` | Synthetic backend для E2E и пользовательские deployment scenarios |
| `docs/deployment/*.md`, общие `docs/*.md` | Фактическое описание readiness, safety, recovery, тестов и этапа |

Доменный job state не зависит от PostgreSQL/Wails. Core не передаёт package bytes, DB handles или идентификаторы адаптера в deployment provider, кроме данных, необходимых синтетическому тесту. JSON-RPC SDK и существующие adapter manifests не получают install RPC/capability.

## Task 1: Job domain и переходы state machine

**Файлы:** создать `internal/core/domain/deployment.go`, `internal/core/domain/deployment_test.go`.

- [x] **Step 1: Написать failing tests** для enum значений, terminal states и разрешённых/запрещённых переходов.

```go
func TestDeploymentTransitionRejectsIllegalState(t *testing.T) {
	if err := ValidateDeploymentTransition(DeploymentStateQueued, DeploymentStateSimulating); err == nil {
		t.Fatal("queued → simulating must require preflight")
	}
}
```

- [x] **Step 2: Запустить** `go test ./internal/core/domain -run 'TestValidateDeploymentTransition|TestDeploymentStateTerminal' -count=1`; ожидаемый compile failure с отсутствующими deployment symbols подтверждён.
- [x] **Step 3: Реализовать** `DeploymentMode` (`test`, `production`), `DeploymentState` (`queued`, `preflighting`, `blocked`, `preflight-passed`, `simulating`, `verifying`, `simulated`, `failed`, `cancelled`), `DeploymentCheckStatus`, `DeploymentJob`, `DeploymentEvent`, `DeploymentPreflight`, `DeploymentRequest` и `ValidateDeploymentTransition(from,to)`.
- [x] **Step 4: Добавить table-driven tests** на разрешённый путь `queued → preflighting → preflight-passed → simulating → verifying → simulated`, ветку blocked, terminal-state запреты и `failed`/`cancelled` только из нетерминального состояния; команда `go test ./internal/core/domain -run 'TestValidateDeploymentTransition|TestDeploymentStateTerminal' -count=1` прошла.
- [x] **Step 5: Commit** `feat: define deployment job domain`.

## Task 2: Package integrity verifier

**Файлы:** создать `internal/deployment/package_verifier.go`, `internal/deployment/package_verifier_test.go`.

- [x] **Step 1: Написать tests** для пакета от `localization.BuildPackage`, hash mismatch, изменённого checksum, corrupt ZIP, duplicate/traversal entry, unknown entry, oversized entry, `oem_firmware=true`, invalid schema и `validation.json` с `Valid=false`.
- [x] **Step 2: Запустить** `go test ./internal/deployment -run TestVerifyLanguagePackage -count=1`; первоначально получил ожидаемый compile failure до реализации `VerifyLanguagePackage` и `MaxDeploymentPackageBytes`.
- [x] **Step 3: Объявить контракт**:

```go
const MaxDeploymentPackageBytes = 16 << 20

type VerifiedPackage struct {
	SHA256 string
	Manifest PackageManifest
}

func VerifyLanguagePackage(artifact []byte, expectedSHA256 string) (VerifiedPackage, error)
```

- [x] **Step 4: Реализовать verifier**: сначала проверить archive size и внешний SHA-256; ограничить число entries и суммарный uncompressed size; разрешать ровно `manifest.json`, `validation.json`, один `resources/<basename>` и `checksums.sha256`; отклонять дубли, абсолютные/parent paths и неизвестные файлы; строго декодировать manifest/report; требовать `carvion.language-package/v1`, `carvion-localization`, `oem_firmware=false`, `Valid=true`; проверить CRC при чтении и все entry checksums.
- [x] **Step 5: Запустить** полный `go test ./internal/deployment -count=1`; тесты успешного синтетического пакета и отказов для повреждённых/небезопасных ZIP прошли.
- [x] **Step 6: Commit** `feat: verify deployment package integrity`.

## Task 3: PostgreSQL migration и audit repository

**Файлы:** создать `db/migrations/00004_deployment_engine.sql`, `internal/core/infrastructure/postgres/deployment_repository.go`, integration tests в `internal/core/infrastructure/postgres/deployment_repository_integration_test.go`.

- [x] **Step 1: Написать integration tests** под `//go:build integration` для создания job, строгого порядка events, append-only constraint, atomic transition/event persistence, blocked production job и повторного чтения после открытия нового repository.
- [x] **Step 2: Создать migration**: `deployment_jobs` с FK на `localization_projects ON DELETE RESTRICT`, package SHA-256, `test|production` mode, target profile/provider IDs nullable, constrained state/result и timestamps; `deployment_job_events` с `(job_id, sequence)` PK, FK `ON DELETE RESTRICT`, status/stage/message/time; индексы на jobs; PK `(job_id, sequence)` покрывает чтение событий в нужном порядке; trigger запрещает UPDATE/DELETE events.
- [x] **Step 3: Добавить interface и repository**:

```go
type DeploymentRepository interface {
	CreateDeploymentJob(context.Context, domain.DeploymentJob) (domain.DeploymentJob, error)
	TransitionDeploymentJob(context.Context, domain.DeploymentJob, domain.DeploymentEvent) error
	AppendDeploymentEvent(context.Context, string, domain.DeploymentState, domain.DeploymentEvent) (domain.DeploymentEvent, error)
	GetDeploymentJob(context.Context, string) (domain.DeploymentJob, error)
	ListDeploymentJobs(context.Context, int) ([]domain.DeploymentJobSummary, error)
}
```

`TransitionDeploymentJob` принимает целевую snapshot-модель, чтобы в той же транзакции записывать preflight/result/recovery вместе с новым state и событием; репозиторий сверяет immutable поля и блокирует строку job.

- [x] **Step 4: Реализовать** pgx transactions: `TransitionDeploymentJob` блокирует job row, валидирует ожидаемый state, обновляет job и добавляет следующую event sequence в той же транзакции; API не предоставляет update/delete для event rows.
- [x] **Step 5: Проверить** `go test -tags=integration ./internal/core/infrastructure/postgres -run '^$' -count=1` для compile-only; также выполнен обычный `go test ./internal/core/infrastructure/postgres -count=1`. `CARVION_TEST_DATABASE_URL` отсутствует, поэтому PostgreSQL integration against migration 00004 не запускался.
- [x] **Step 6: Commit** `feat: persist deployment audit jobs`.

## Task 4: Provider capability contract и test provider

**Файлы:** создать `internal/core/application/deployment_provider.go`, `internal/deployment/simulated_provider.go`, соответствующие unit tests.

- [x] **Step 1: Написать tests** на provider `mode=simulation`, `CanInstall=false`, capability раздельно сообщает result verification/recovery check; исполнение simulation не меняет оборудование и заканчивается только `simulated`.
- [x] **Step 2: Реализовать контракт**:

```go
type DeploymentProviderCapabilities struct {
	Mode                    domain.DeploymentProviderMode
	CanInstall              bool
	CanVerifyResult         bool
	CanCheckRecovery        bool
	CanSimulate             bool
}

type DeploymentProvider interface {
	ID() string
	Capabilities() DeploymentProviderCapabilities
	Simulate(context.Context, domain.DeploymentJob, []byte, func(domain.DeploymentEvent) error) error
	VerifyResult(context.Context, domain.DeploymentJob, string) (domain.DeploymentResultVerification, error)
	CheckRecovery(context.Context, domain.DeploymentJob) (domain.RecoveryCapabilityResult, error)
}
```

- [x] **Step 3: Добавить synthetic implementation** только для embedded simulator profiles с `synthetic=true`; она испускает ограниченный упорядоченный event sequence, верифицирует test package hash/result, умеет сообщить только `simulated recovery available`, не имеет OS/device/DB/file/network API и не определяет `Install` method.
- [x] **Step 4: Запустить** `go test ./internal/deployment ./internal/core/application -run 'DeploymentProvider|Simulat' -count=1` и полный `go test ./internal/deployment -count=1`; оба прошли.
- [x] **Step 5: Commit** `feat: add synthetic deployment provider`.

## Task 5: Core preflight и job orchestration

**Файлы:** создать `internal/core/application/deployment.go`, `internal/core/application/deployment_test.go`; изменить `internal/core/application/service.go` и конструкторы сервиса.

- [x] **Step 1: Написать failing application tests** для package verifier success/failure; exact synthetic supported; unknown/unsupported rejection; Production blocked for каждый readiness prerequisite; provider не вызывается для Production; provider installation/result/recovery capability gates; event persisted before callback; provider error, deadline, cancellation and recovery check. Встроенный simulator catalog содержит только synthetic профили, поэтому non-synthetic rejection остаётся защитной runtime-проверкой без такого fixture.
- [x] **Step 2: Определить** `productionDeploymentEnabled = false` в Core policy; mode `production` записывает job/checklist `blocked` без вызова provider. Policy compile-time, config/UI flag её не переопределяет.
- [x] **Step 3: Реализовать** `DeploymentService` с `StartJob`, `RunJob`, `CancelJob`, `GetJob`, `ListJobs`; `StartJob` получает CARVION package через `LocalizationService`, сохраняет hash, а `RunJob` перечитывает именно сохранённый artifact по SHA, выполняет integrity verification и preflight. Только test mode + embedded synthetic `supported` profile + synthetic provider `CanSimulate && CanVerifyResult && CanCheckRecovery` запускает simulation.
- [x] **Step 4: Гарантировать порядок** для progress: Core валидирует stage/state → repository атомарно сохраняет событие/переход → publisher best-effort emit. UI может восстановить пропущенные события через `GetJob`/`ListJobs`; ошибка хранения останавливает provider flow.
- [x] **Step 5: Проверить** `go test ./internal/core/application ./internal/composition ./internal/core/infrastructure/postgres ./internal/localization ./internal/deployment -count=1`, `go test -tags=integration ./internal/core/infrastructure/postgres -run '^$' -count=1`, `go vet` этих Go-пакетов и `git diff --check`; все завершились с кодом 0. Отдельно проверены production blocker, provider error, timeout, cancel и восстановление. PostgreSQL DB integration пропущен: `CARVION_TEST_DATABASE_URL` отсутствует.
- [x] **Step 6: Подключить** `DeploymentService` через `application.Service` и local composition root; `internal/composition/local.go` инжектирует `postgres.NewDeploymentRepository(pool)`, `deployment.NewPackageVerifier()` и только `deployment.NewSimulatedProvider()`.
- [x] **Step 7: Commit** `feat: add fail-closed deployment core`.

## Task 6: Wails facade, live progress и lifecycle

**Файлы:** изменить `desktop/application_service.go`, `desktop/application_service_test.go`, `desktop/main.go`; regenerate `desktop/frontend/bindings/` установленным Wails v3.0.0.

- [x] **Step 1: Написать facade tests** на list/start/get/cancel, context propagation, duplicate active ID rejection, progress event payload и shutdown cancellation.
- [x] **Step 2: Реализовать** `deploymentCore` boundary; Wails `StartDeploymentJob` возвращает persistent job, worker выполняет Core orchestration асинхронно; `GetDeploymentJob` перечитывает committed events, `CancelDeploymentJob` передаёт отмену активному run.
- [x] **Step 3: Добавить active job lifecycle map** в service, отменять jobs при `ServiceShutdown`, очищать entry при любом завершении; Core timeout bounded двумя минутами и context-aware.
- [x] **Step 4: Регистрировать/эмитировать Wails event** `deployment:progress` в Wails v3.0.0; payload содержит job ID и persisted sequence. Для unit tests используется injected publisher вместо глобального Wails runtime.
- [x] **Step 5: Генерировать bindings** командой `wails3 generate bindings -ts .`; desktop Go tests, facade/simulator race tests и `go vet ./...` завершились с кодом 0. Go tests запускались при временном embed fixture, пока frontend dist ещё не собран; Task 7 выполнит настоящую сборку frontend.
- [x] **Step 6: Commit** `feat: expose deployment jobs to Wails`.

## Task 7: Deployment Desktop screen и E2E

**Файлы:** создать `desktop/frontend/src/features/deployment-page.tsx`; изменить `navigation.ts`, `App.tsx`, `lib/backend.ts`, `e2e-entry.tsx`, `e2e/studio.spec.ts`.

- [x] **Step 1: Добавить E2E scenarios**: Production shows blockers and persisted blocked result; valid approved project + synthetic supported profile runs simulated lifecycle; final UI explicitly says no vehicle changed; invalid package check fails without simulation; RPC/provider failure, cancel and reloaded job history states.
- [x] **Step 2: Запустить** `cd desktop/frontend && npm test -- --grep 'deployment'`; expected red confirmed before adding workspace.
- [x] **Step 3: Реализовать typed backend** for list/start/get/cancel and `onDeploymentProgress`; Wails implementation subscribes using `@wailsio/runtime Events.On`, test backend publishes explicitly marked fixtures.
- [x] **Step 4: Реализовать UI**: project and embedded synthetic target selectors; separate `Test` and `Production blocked` states; cards for package integrity, compatibility, installation capability, recovery and bench evidence; live progress and persisted jobs; cancel only active test; no generic install button and no wording `installed/deployed`.
- [x] **Step 5: Запустить** `npm run typecheck`, all Playwright tests with `npm test`, and production frontend `npm run build`; all passed after aligning with regenerated Wails bindings.
- [x] **Step 6: Commit** `feat: add deployment workspace` (`08d83bf`).

## Task 8: Deployment docs and project records

**Файлы:** создать `docs/deployment/ARCHITECTURE.md`, `PREFLIGHT.md`, `SAFETY.md`, `RECOVERY.md`, `TESTING.md`; обновить docs indexes and state.

- [x] **Step 1: Документировать** audit findings, mode-specific gates, provider capability contract, full state machine, package integrity vs authenticity, append-only local audit limits and current absence of any hardware/bench validation.
- [x] **Step 2: Обновить** `docs/README.md`, `PROJECT.md`, `architecture.md`, `ROADMAP.md`, `PROGRESS.md`, `CHANGELOG.md`, `DECISIONS.md`, `TESTING.md`, `KNOWN_ISSUES.md`, `NEXT_STEPS.md`, `DESKTOP.md`, `TECH_STACK.md` and adapter/localization notes only where their existing descriptions change.
- [x] **Step 3: Проверить** each statement against manifest/capabilities, Core production policy, migration, package verifier and current E2E results. Mark hardware operation as not tested; mark only synthetic provider as implemented/verified.
- [x] **Step 4: Запустить** `git diff --check` and resolve every reported whitespace error.
- [x] **Step 5: Commit** `docs: document deployment engine readiness and safety`.

## Task 9: Full verification and completion record

- [x] **Step 1:** Run `make desktop-test` (root/Desktop Go, TypeScript, Playwright, Vite): passed; 18 Playwright scenarios.
- [x] **Step 2:** Run `make lint` and `make build` (API and both adapters): passed.
- [x] **Step 3:** Run `make desktop-build`; verify arm64 Mach-O and `codesign --verify --deep --strict desktop/bin/carvionstudio.app`: passed.
- [x] **Step 4:** Applied Goose migrations 00001–00004 to a dedicated temporary PostgreSQL 18 container and ran `CARVION_TEST_DATABASE_URL=… go test -tags=integration ./... -count=1`: passed. The container was removed; the developer database was not used.
- [x] **Step 5:** Verified `git diff --check`, formatted changed Go test, reconciled `docs/PROGRESS.md` and `docs/TESTING.md`, and verified the worktree after commits. No physical hardware test was run; Production Mode remains blocked.
- [x] **Step 6:** Committed final docs and verification reconciliation with `docs: document deployment engine readiness and safety`.

## Self-review: design coverage

| Approved spec item | Plan task |
|---|---|
| Readiness audit, fail-closed Production Mode, unsupported target blocks | Tasks 1, 5, 7, 8 |
| Deployment Job and strict state machine | Tasks 1, 3, 5 |
| Package integrity | Task 2 |
| Separate install/result/recovery capabilities and synthetic provider | Task 4 |
| Progress events, lifecycle, cancellation | Tasks 5–7 |
| PostgreSQL persistent job/event audit | Task 3 |
| Result verification and failure/recovery checks | Tasks 4–5 |
| Desktop preflight, results, blockers and user messaging | Task 7 |
| Required docs and actual verification record | Tasks 8–9 |

## Self-review: plan quality

- No unresolved placeholders or `TODO` steps remain.
- Types referenced by later tasks are introduced in earlier tasks.
- No SDK install RPC, HTTP route, DB credential, arbitrary path input, vehicle IO or real hardware test is added.
- PostgreSQL integration is conditional on an explicitly dedicated test database; compile-only is not described as an executed DB test.
