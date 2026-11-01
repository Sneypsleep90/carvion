# CARVION Simulator Step 5 Implementation Plan

> **For agentic workers:** Execute this plan task-by-task with the `executing-plans` skill. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Run deterministic synthetic infotainment profiles through the existing JSON-RPC adapter boundary and complete a verified localization package workflow in CARVION Studio without a vehicle.

**Architecture:** `internal/simulator` is the single source of synthetic profiles, scenarios and resource fixtures. `adapter-android-demo` consumes those fixtures through the existing SDK; Core orchestrates adapter discovery, detection, compatibility, localization and package verification through application services. Wails exposes the workflow directly to the new Simulator screen; no HTTP routes, database migration or persistent global operation history are added.

**Tech Stack:** Existing Go Core, Adapter SDK v1, Plugin Manager JSON-RPC subprocesses, pgx/PostgreSQL Localization Engine, Wails 3 bindings, React/TypeScript and Playwright.

---

### Task 1: Synthetic profile catalog and fixtures

**Files:** Create `internal/simulator/profile.go`, `internal/simulator/profile_test.go`, `internal/simulator/testdata/profiles.json`, `internal/simulator/testdata/resources-zh-CN.xml`.

- [x] **Step 1: Write failing tests** for required profile IDs, explicit synthetic markers, complete identity/hardware/firmware, resource fixture UTF-8 and defensive copies.
- [x] **Step 2: Run `go test ./internal/simulator -count=1`** and confirm the new behavior fails before implementation.
- [x] **Step 3: Implement the minimal immutable profile catalog** with Platform A/B, Unknown Firmware, Incompatible Hardware and simulated communication failure profiles.
- [x] **Step 4: Re-run `go test ./internal/simulator -count=1`** and confirm all catalog tests pass.

### Task 2: Multi-profile JSON-RPC demo adapter

**Files:** Modify `internal/adapters/androiddemo/adapter.go`, `internal/adapters/androiddemo/adapter_test.go`, `cmd/adapter-android-demo/main.go`, `adapters/adapter-android-demo/manifest.json` only if required by existing contracts.

- [x] **Step 1: Add failing tests** for detecting both supported profiles, unknown identity, exact supported firmware, unknown firmware, incompatible hardware and the simulated RPC error.
- [x] **Step 2: Run `go test ./internal/adapters/androiddemo -count=1`** and confirm the profile scenarios fail.
- [x] **Step 3: Reuse `internal/simulator` fixtures** in the adapter; preserve current SDK v1 and strict exact-build compatibility behavior.
- [x] **Step 4: Re-run adapter tests** and verify XML inspection still returns synthetic Chinese resources with existing parsing guarantees.

### Task 3: Core simulator orchestration

**Files:** Create `internal/core/domain/simulator.go`, `internal/core/application/simulator.go`, `internal/core/application/simulator_test.go`; modify `internal/core/application/service.go` and `internal/composition/local.go` as needed.

- [x] **Step 1: Write failing use-case tests** for capability-based adapter selection, successful ordered events, unknown/unsupported early exits, RPC error/context cancellation and package integrity validation.
- [x] **Step 2: Run the focused application tests** and confirm the simulator workflow is not implemented.
- [x] **Step 3: Implement Core orchestration** using `AdapterGateway`, existing Localization use cases and the synthetic profile catalog; keep adapter choice, statuses and failures out of frontend logic.
- [x] **Step 4: Verify package bytes** by SHA-256, ZIP readability, resource entry and non-OEM manifest fields before returning a successful run.
- [x] **Step 5: Re-run Core tests** and confirm no project/package is created for unknown or incompatible profiles.

### Task 4: Wails service boundary

**Files:** Modify `desktop/application_service.go`, `desktop/application_service_test.go`, regenerate `desktop/frontend/bindings/` with Wails v3.0.0.

- [x] **Step 1: Add failing facade tests** for profile listing, run delegation, context propagation and structured error return.
- [x] **Step 2: Implement Wails methods** over the Core simulator use cases; preserve degraded startup and lifecycle behavior.
- [x] **Step 3: Regenerate bindings** and run `cd desktop && go test ./...`.

### Task 5: Simulator Mode UI and E2E

**Files:** Modify `desktop/frontend/src/navigation.ts`, `src/App.tsx`, `src/lib/backend.ts`, `e2e-entry.tsx`, `e2e/studio.spec.ts`; create `src/features/simulator-page.tsx`.

- [x] **Step 1: Add failing Playwright scenarios** for supported full cycle, unknown firmware, incompatible hardware and simulated communication failure.
- [x] **Step 2: Implement Simulator screen** with explicit `SIMULATION` marker, profile selection, run controls, live-result states, event/operation log, package verification and download.
- [x] **Step 3: Use only Wails bindings in production UI**; keep the Playwright backend synthetic and clearly isolated to E2E.
- [x] **Step 4: Run `cd desktop/frontend && npm run typecheck && npm test && npm run build`** and confirm the end-to-end scenarios pass. `make desktop-test` passed with all 10 Playwright E2E tests after Core cancellation event normalization.

### Task 6: Documentation and final verification

**Files:** Create `docs/simulator/OVERVIEW.md`, `SCENARIOS.md`, `TEST_RESULTS.md`; update `docs/README.md`, `docs/architecture.md`, `docs/PROJECT.md`, `docs/DESKTOP.md`, `docs/ROADMAP.md`, `docs/PROGRESS.md`, `docs/CHANGELOG.md`, `docs/TESTING.md`, `docs/KNOWN_ISSUES.md`, `docs/NEXT_STEPS.md` and root `README.md` as applicable.

- [x] Document profile provenance, event lifecycle, statuses, faults, package verification and explicit non-vehicle/non-firmware guarantees.
- [x] Run `make desktop-test`, `make lint`, `make build`, `make desktop-build` and uncached integration tests on a dedicated migrated PostgreSQL test database.
- [x] Run relevant race tests and `git diff --check`; record exact successful commands and remaining limitations.
- [x] Re-read the plan and docs against the final implementation; all documented synthetic scenarios are covered by unit, process, PostgreSQL or UI tests as described in `docs/simulator/TEST_RESULTS.md`.
