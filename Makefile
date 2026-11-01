GO ?= go
DOCKER_COMPOSE ?= docker compose
GOOSE_VERSION ?= v3.29.0
GOOSE_BUILD_TAGS ?= no_mysql no_sqlite3 no_ydb no_azuresql no_clickhouse no_libsql no_mssql no_vertica
WAILS3 ?= $(shell go env GOPATH)/bin/wails3

.PHONY: help dev db-up db-down migrate-up migrate-down test test-integration lint build build-api build-adapter desktop-dev desktop-test desktop-build desktop-run

help:
	@printf '%s\n' \
		'make dev             Run the API using settings from .env' \
		'make db-up           Start local PostgreSQL in Docker Compose' \
		'make db-down         Stop PostgreSQL and preserve its volume' \
		'make migrate-up      Apply all PostgreSQL migrations' \
		'make migrate-down    Roll back the most recent migration' \
		'make test            Run unit tests' \
		'make test-integration Run PostgreSQL integration tests' \
		'make lint            Run gofmt checks and go vet' \
		'make build           Build API and adapter-android-demo' \
		'make desktop-dev     Run CARVION Studio with Wails hot reload' \
		'make desktop-test    Test Go Desktop service and React frontend' \
		'make desktop-build   Build and package CARVION Studio for macOS' \
		'make desktop-run     Build and launch the packaged macOS app'

dev:
	@test -f .env || { echo 'Copy .env.example to .env first.' >&2; exit 1; }
	@set -eu; set -a; . ./.env; set +a; $(GO) run ./cmd/carvion-api

db-up:
	$(DOCKER_COMPOSE) up -d db

db-down:
	$(DOCKER_COMPOSE) down

migrate-up:
	@test -f .env || { echo 'Copy .env.example to .env first.' >&2; exit 1; }
	@set -eu; set -a; . ./.env; set +a; $(GO) run -tags='$(GOOSE_BUILD_TAGS)' github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION) -dir db/migrations postgres "$$DATABASE_URL" up

migrate-down:
	@test -f .env || { echo 'Copy .env.example to .env first.' >&2; exit 1; }
	@set -eu; set -a; . ./.env; set +a; $(GO) run -tags='$(GOOSE_BUILD_TAGS)' github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION) -dir db/migrations postgres "$$DATABASE_URL" down

test:
	$(GO) test ./...

test-integration:
	@test -n "$$CARVION_TEST_DATABASE_URL" || { echo 'Set CARVION_TEST_DATABASE_URL to a test database URL.' >&2; exit 1; }
	$(GO) test -tags=integration ./...

lint:
	@files="$$(find . -name '*.go' -type f -not -path './vendor/*' -not -path './desktop/build/*')"; \
	if [ -n "$$files" ]; then \
		unformatted="$$(gofmt -l $$files)"; \
		if [ -n "$$unformatted" ]; then printf '%s\n' "$$unformatted"; exit 1; fi; \
	fi
	$(GO) vet ./...

build: build-api build-adapter

build-api:
	mkdir -p bin
	$(GO) build -o bin/carvion-api ./cmd/carvion-api

build-adapter:
	mkdir -p adapters/adapter-android-demo/bin
	$(GO) build -o adapters/adapter-android-demo/bin/adapter-android-demo ./cmd/adapter-android-demo

desktop-dev: build-adapter
	@test -f .env || { echo 'Copy .env.example to .env first.' >&2; exit 1; }
	@set -eu; set -a; . ./.env; set +a; \
		export ADAPTERS_DIR="$(CURDIR)/adapters"; \
		export PATH="$(shell go env GOPATH)/bin:$$PATH"; \
		cd desktop && $(WAILS3) dev

desktop-test:
	$(GO) test ./...
	cd desktop && MACOSX_DEPLOYMENT_TARGET=13.0 CGO_CFLAGS='-mmacosx-version-min=13.0' CGO_LDFLAGS='-mmacosx-version-min=13.0' $(GO) test ./...
	cd desktop/frontend && npm run typecheck && npm test && npm run build

desktop-build: build-adapter
	cd desktop && PATH="$(shell go env GOPATH)/bin:$$PATH" $(WAILS3) package

desktop-run: desktop-build
	@test -f .env || { echo 'Copy .env.example to .env first.' >&2; exit 1; }
	@set -eu; set -a; . ./.env; set +a; \
		export ADAPTERS_DIR="$(CURDIR)/adapters"; \
		cd desktop && ./bin/carvionstudio.app/Contents/MacOS/carvionstudio
