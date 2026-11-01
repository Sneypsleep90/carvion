# CARVION Foundation Implementation Plan

> **For agentic workers:** Execute the checked steps in order, preserving the request's limited Foundation scope. The project workspace was empty and has no Git metadata, so changes are kept in place without commits.

**Goal:** Build a runnable Go API foundation with PostgreSQL migrations, local Compose setup, tests, CI, and documentation.

**Architecture:** Keep the executable in `cmd/carvion-api` and split configuration, HTTP routing, and PostgreSQL pool setup across the requested `internal` packages. The API exposes health and readiness only; readiness pings PostgreSQL with a short context timeout. Goose SQL migrations create the initial catalog tables.

**Tech Stack:** Go 1.27.3, chi, pgx/v5, goose, PostgreSQL 18, Docker Compose, `slog`, Go testing, GitHub Actions.

---

### Task 1: Configuration and HTTP contracts

**Files:**
- Create: `internal/config/config_test.go`
- Create: `internal/server/handler_test.go`
- Create implementation after red tests: `internal/config/config.go`, `internal/server/handler.go`

- [x] Test configuration defaults, required database settings, and invalid HTTP ports; run `go test ./internal/config` to observe missing implementation.
- [x] Implement environment parsing and validation with defaults `development`, `127.0.0.1`, and `8080`.
- [x] Test `/healthz` JSON/200 and `/readyz` success and unavailable/503 using a minimal pinger test double; run `go test ./internal/server` to observe missing implementation.
- [x] Implement chi routes, bounded readiness ping, JSON responses, and structured request logging.
- [x] Run `gofmt` and `go test ./internal/config ./internal/server`.

### Task 2: Database pool and migrations

**Files:**
- Create: `internal/database/pool_test.go`
- Create: `internal/database/pool.go`
- Create: `db/migrations/00001_create_catalog_tables.sql`

- [x] Add a `//go:build integration` test that opens the configured test database and pings it; verify it compiles with `go test -tags=integration ./internal/database`.
- [x] Implement a context-bounded `pgxpool` connection function that closes the pool on failed startup ping.
- [x] Add goose Up/Down SQL for manufacturers, vehicle models, and infotainment platforms, with UUID/time defaults, foreign keys, and suitable uniqueness/index constraints.
- [x] Format Go files and run database unit/integration checks against PostgreSQL.

### Task 3: API process and local database

**Files:**
- Create: `cmd/carvion-api/main.go`
- Create: `compose.yaml`, `.env.example`, `.gitignore`, `Makefile`

- [x] Wire validated configuration, JSON `slog`, pool startup, HTTP timeouts, signal cancellation, and graceful shutdown.
- [x] Configure PostgreSQL 18 with environment-only credentials, persistent named volume, host port 5432, and a healthcheck.
- [x] Add `dev`, `db-up`, `db-down`, `migrate-up`, `migrate-down`, `test`, `lint`, and `build` Make targets; invoke goose via a pinned `go run` tool version.
- [x] Ensure `.env` and generated binaries are ignored and document local setup in Make help/comments.

### Task 4: CI and documentation

**Files:**
- Create: `.github/workflows/ci.yml`
- Create: `README.md`, `docs/architecture.md`

- [x] Add push and pull request CI for Go setup, formatting, `go vet`, `go test`, and `go build` without database secrets.
- [x] Document setup, migration commands, tests, project structure, roadmap, and the temporary module path replacement.
- [x] Describe implemented and planned architecture components separately.

### Task 5: Verification

- [x] Run `gofmt`, `go vet ./...`, `go test ./...`, and `go build ./cmd/carvion-api`.
- [x] Run Docker Compose, Goose migrations, and endpoint checks.
- [x] Run the integration test against the Compose database.
- [x] Review the file list and verify each requested Foundation criterion.
