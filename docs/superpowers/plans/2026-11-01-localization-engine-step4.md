# Localization Engine Step 4 Implementation Plan

> **For agentic workers:** Use this plan task-by-task. Keep the Core transport-independent and update documentation after the step.

**Goal:** Import Android XML and the documented CARVION JSON schema into persistent localization projects, edit and validate Russian translations, export source resources/CSV, and build reproducible CARVION language ZIP packages.

**Architecture:** Pure parsers, translation validation, resource exporters and deterministic ZIP building live under `internal/localization`. Core domain/application expose project operations through a repository port; pgx implements project, translation, validation history and package persistence. `desktop.ApplicationService` delegates to Core and React uses only generated Wails bindings. No HTTP routes are added.

**Tech Stack:** Existing Go 1.27, standard library (`encoding/xml`, `encoding/json`, `encoding/csv`, `archive/zip`, `crypto/sha256`), pgx v5, Goose SQL migrations, Wails 3, React/TypeScript, Playwright.

---

### Task 1: Resource contracts and parsers

**Files:** `internal/core/domain/localization.go`, `internal/localization/types.go`, `internal/localization/parser.go`, `xml.go`, `json.go`, `csv.go`; focused `*_test.go` files.

- [x] Test UTF-8 rejection, Android strings/string-array/plurals parsing, duplicate resource IDs, non-translatable resources and malformed XML.
- [x] Test JSON schema version/shape, duplicate IDs and metadata retention.
- [x] Test strict CSV headers, quoting, duplicate keys, stale source text and UTF-8.
- [x] Implement parsers with explicit resource kind/group/index/plural metadata and stored source snapshot.
- [x] Export parsed Android XML while retaining the source hierarchy and attributes; export JSON in the documented schema.

### Task 2: Translation validation and application use cases

**Files:** `internal/core/application/ports.go`, `localization.go`; `internal/localization/validator.go`; unit tests.

- [x] Test blank/missing translations, malformed UTF-8, source changes after approval, and Java format placeholders including positional reorder and type/count mismatch.
- [x] Test valid and invalid plural categories and duplicate plural categories.
- [x] Implement draft/approved/needs_review status transitions and deterministic validation reports.
- [x] Add Core operations for create/import, update, approve, list/get, CSV import/export, validate and source export.
- [x] Ensure reimport marks previously approved translations `needs_review` when their source text changes.

### Task 3: PostgreSQL repository and migration

**Files:** `db/migrations/00003_localization_engine.sql`, `internal/core/infrastructure/postgres/localization_repository.go`, integration tests, composition wiring.

- [x] Test project/resource/translation round-trip, transaction rollback, append-only history, stale approval transition and validation/package persistence.
- [x] Add FK/check/unique constraints and indexes for project resources, locale translations, revision history, validation runs and language-package artifacts.
- [x] Implement pgx-only repository methods with parameterized SQL and explicit transactions; do not add an ORM.
- [x] Wire the repository into `composition.NewLocal` and application service construction.

### Task 4: Reproducible language package builder

**Files:** `internal/localization/package.go`, package-builder tests.

- [x] Test identical inputs produce byte-identical ZIP bytes and SHA-256.
- [x] Test package file list, checksums, validation report and explicit non-OEM manifest fields.
- [x] Test invalid projects and unapproved translations cannot produce a package.
- [x] Build ZIP using sorted paths, fixed timestamps, stable JSON, source resource export, `manifest.json`, `validation.json` and `checksums.sha256`.
- [x] Persist package bytes and metadata via the repository.

### Task 5: Wails facade and Localization UI

**Files:** `desktop/application_service.go`, generated bindings, `desktop/frontend/src/lib/backend.ts`, `src/features/localization-page.tsx`, UI helpers and `e2e/studio.spec.ts`.

- [x] Test Wails facade delegates project workflow methods to Core and preserves context/errors.
- [x] Test UI file import, project/resource display, search, editing, validation, approval, CSV actions, resource export and package download using a synthetic backend.
- [x] Replace current read-only inspection page with project workflow and keep all business rules in Go Core.
- [x] Regenerate TypeScript bindings with the installed Wails v3.0.0 CLI.

### Task 6: Documentation and full verification

**Files:** `docs/localization/ENGINE.md`, `FORMATS.md`, `VALIDATION.md`, `PACKAGES.md`; `docs/PROGRESS.md`, `CHANGELOG.md`, `TESTING.md`, `NEXT_STEPS.md`, project/architecture/roadmap and README indexes.

- [x] Document exact XML/JSON/CSV formats, lifecycle/status rules, validation and ZIP provenance.
- [x] Run root Go tests, race tests where relevant, PostgreSQL integration tests on a dedicated test database, Desktop Go tests, TypeScript/E2E/build, lint and Wails package build.
- [x] Record only checks actually run and retain existing history.
