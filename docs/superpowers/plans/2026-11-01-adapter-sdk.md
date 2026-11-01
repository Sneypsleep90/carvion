# Vehicle Adapter SDK Implementation Plan

> **For agentic workers:** Use subagent-driven-development or executing-plans to implement this plan task-by-task. The user's explicit request authorizes inline execution. This workspace has no Git metadata, so commits are unavailable.

**Goal:** Add a versioned adapter SDK, trusted local process manager, synthetic Android demo adapter, compatibility schema, and public adapter API routes.

**Architecture:** SDK contracts live in `pkg/adaptersdk/v1`. The manager validates local manifests and invokes one adapter process per JSON-RPC 2.0 line-framed call, passing bounded request payloads and no database or device handles. The API depends on the manager, while the demo adapter parses only core-supplied JSON/XML contents and uses synthetic profiles.

**Tech Stack:** Go 1.27.3 standard library, existing chi/pgx/goose stack, JSON-RPC 2.0 over stdin/stdout, PostgreSQL SQL migrations.

---

### Task 1: Versioned SDK contract

**Files:**
- Create: `pkg/adaptersdk/v1/types.go`, `rpc.go`, `rpc_test.go`

- [x] Define manifest, identity, firmware, hardware, compatibility, capabilities, resource, and operation request/response types with SDK API `v1` and explicit `supported`, `unsupported`, `unknown` constants.
- [x] Write tests first for one-request JSON-RPC dispatch, unknown method, invalid parameters, and adapter errors; run `go test ./pkg/adaptersdk/v1` to observe missing definitions.
- [x] Implement the standard-library JSON-RPC server with a 1 MiB frame limit and one request/response per process invocation.
- [x] Run `gofmt` and `go test ./pkg/adaptersdk/v1`.

### Task 2: Synthetic Android demo adapter

**Files:**
- Create: `internal/adapters/androiddemo/adapter.go`, `adapter_test.go`
- Create: `internal/adapters/androiddemo/testdata/profiles.json`, `vehicle.json`, `strings.xml`
- Create: `cmd/adapter-android-demo/main.go`
- Create: `adapters/adapter-android-demo/manifest.json`

- [x] Write tests first for synthetic JSON detection, exact firmware match, unknown build and vehicle outcomes, and XML string extraction.
- [x] Implement the adapter using embedded synthetic profiles; parse JSON/XML payloads supplied by the caller, never local paths or device connections.
- [x] Add the executable entry point using the SDK server and the matching v1 manifest.
- [x] Run `go test ./internal/adapters/androiddemo ./cmd/adapter-android-demo`.

### Task 3: Trusted local plugin manager

**Files:**
- Create: `internal/pluginmanager/manager.go`, `manifest.go`, `rpc.go`, `manager_test.go`

- [x] Test valid discovery, malformed and incompatible manifests, duplicate IDs, path escape rejection, process RPC errors/termination, and deadline cancellation.
- [x] Implement direct-child local discovery, bounded manifest parsing, API/manifest version checks, executable containment checks, in-memory registration, one-shot process launch, bounded JSON-RPC frames, sanitized child environment, and process cleanup.
- [x] Run `go test ./internal/pluginmanager`.

### Task 4: Compatibility schema migration

**Files:**
- Create: `db/migrations/00002_adapter_sdk.sql`

- [x] Add adapter definitions/versions, hardware variants, firmware builds, and compatibility rules. Composite foreign keys must enforce that a rule's platform, hardware variant, and firmware build belong together; status must be constrained to the SDK states.
- [x] Add uniqueness and foreign-key indexes, then verify Goose Up/Down/Up against the local PostgreSQL database.

### Task 5: HTTP API and configuration wiring

**Files:**
- Modify: `internal/config/config.go`, `internal/config/config_test.go`
- Modify: `internal/server/handler.go`, `internal/server/handler_test.go`
- Modify: `cmd/carvion-api/main.go`

- [x] Add optional `ADAPTERS_DIR` with default `./adapters` and unit coverage.
- [x] Add tests for adapter listing, validated compatibility requests, unknown adapters, plugin failures, and structured JSON errors; run them red before handler implementation.
- [x] Implement `GET /api/v1/adapters` and `POST /api/v1/compatibility/check` with bounded JSON bodies, strict decoding, and no administrative routes.
- [x] Discover and register local plugins during API startup; keep health/readiness and shutdown behavior intact.
- [x] Run `go test ./internal/config ./internal/server ./cmd/carvion-api`.

### Task 6: Build, CI, documentation, and verification

**Files:**
- Modify: `Makefile`, `.gitignore`, `.github/workflows/ci.yml`, `README.md`, `docs/architecture.md`

- [x] Extend `make build` to build both the API and demo adapter; ignore only generated plugin binary output.
- [x] Ensure CI tests and builds all Go packages without PostgreSQL or credentials.
- [x] Document SDK v1, trust boundaries, JSON-RPC lifecycle, local plugin setup, compatibility schema, and synthetic-only data.
- [x] Run `make lint`, `make test`, `make build`, real migration and API smoke checks when Docker is available; review final files and scope.
