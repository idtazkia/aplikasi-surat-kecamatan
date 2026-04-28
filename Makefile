# Aplikasi Surat Kecamatan — Makefile
#
# Target group:
#   - Tooling:    install-tools
#   - DB:         migrate-up, migrate-down, seed-demo, reset-demo, sqlc
#   - Backend:    build, dev, test, coverage, lint
#   - Frontend:   web-install, web-dev, web-build, web-test, web-lint
#   - Concepts:   concepts-inject, concepts-build, concepts-lint, concepts-serve
#   - Misc:       clean, help

SHELL := /bin/bash
GO ?= go
NPM ?= npm

# Database connection (override via env atau .env)
DATABASE_URL ?= postgres://surat:surat@localhost:5432/surat_dev?sslmode=disable

# Migration directories + goose version tables (terpisah supaya schema dan seed
# tidak konflik di version_id — keduanya pakai prefix 0001_*.sql)
MIGRATIONS_SCHEMA := db/migrations/schema
MIGRATIONS_SEED   := db/migrations/demo-seed
GOOSE_SEED_TABLE  := goose_demo_seed_version

# Tool versions (lock untuk reproducibility)
GOOSE_VERSION  := v3.22.1
SQLC_VERSION   := v1.27.0
MDBOOK_VERSION := v0.4.40

.DEFAULT_GOAL := help

## help: tampilkan daftar target
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /' | sort

## install-tools: install goose, sqlc, mdbook ke $$GOPATH/bin atau ~/.cargo/bin
install-tools:
	$(GO) install github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION)
	$(GO) install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
	@command -v mdbook >/dev/null 2>&1 || cargo install mdbook --vers $(subst v,,$(MDBOOK_VERSION)) --locked
	@echo "Tools installed. Run 'make help' for available targets."

## migrate-up: apply schema migration ke DATABASE_URL
migrate-up:
	goose -dir $(MIGRATIONS_SCHEMA) postgres "$(DATABASE_URL)" up

## migrate-down: rollback satu schema migration
migrate-down:
	goose -dir $(MIGRATIONS_SCHEMA) postgres "$(DATABASE_URL)" down

## migrate-status: status schema migration
migrate-status:
	goose -dir $(MIGRATIONS_SCHEMA) postgres "$(DATABASE_URL)" status

## seed-demo: apply demo-seed migration (env demo/dev saja, JANGAN di production)
seed-demo:
	goose -dir $(MIGRATIONS_SEED) -table $(GOOSE_SEED_TABLE) postgres "$(DATABASE_URL)" up

## reset-demo: rollback semua seed lalu re-apply (schema tidak disentuh)
reset-demo:
	goose -dir $(MIGRATIONS_SEED) -table $(GOOSE_SEED_TABLE) postgres "$(DATABASE_URL)" down-to 0
	goose -dir $(MIGRATIONS_SEED) -table $(GOOSE_SEED_TABLE) postgres "$(DATABASE_URL)" up

## sqlc: generate Go code dari SQL queries
sqlc:
	sqlc generate

## build: build server binary ke bin/server
build:
	$(GO) build -o bin/server ./cmd/server

## dev: run backend dengan auto-reload (butuh air atau go run loop)
dev:
	$(GO) run ./cmd/server

# Filter out web/node_modules (third-party non-app Go files muncul lewat ./...).
GO_PACKAGES = $(shell $(GO) list ./... | grep -v '/web/')

## test: run Go tests
test:
	$(GO) test -race -count=1 $(GO_PACKAGES)

## coverage: run tests dengan coverage report
coverage:
	$(GO) test -race -coverprofile=coverage.txt -covermode=atomic $(GO_PACKAGES)
	$(GO) tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## web-install: install dependencies frontend
web-install:
	cd web && $(NPM) install

## web-dev: run Vite dev server
web-dev:
	cd web && $(NPM) run dev

## web-build: production build frontend
web-build:
	cd web && $(NPM) run build

## web-test: run frontend unit tests (Vitest, dengan coverage gate 80%)
web-test:
	cd web && $(NPM) run test:unit

## web-test-e2e: run end-to-end tests (Playwright + testcontainers)
##               Butuh Docker running. Otomatis spin up PostgreSQL, build Go
##               backend, start frontend dev server.
web-test-e2e:
	cd web && $(NPM) run test:e2e

## web-test-all: unit + E2E
web-test-all: web-test web-test-e2e

## web-lint: run frontend linter
web-lint:
	cd web && $(NPM) run lint

## concepts-inject: scan marker di source, inject permalink ke concept page markdown
concepts-inject:
	$(GO) run ./tools/concept-links inject

## concepts-emit: generate concept-links.json untuk Vue student drawer
concepts-emit:
	$(GO) run ./tools/concept-links emit-json > docs/concepts/src/concept-links.json

## concepts-lint: deteksi orphan marker / @anchor
concepts-lint:
	$(GO) run ./tools/concept-links lint

## concepts-build: full concept catalog build (inject + mdbook)
concepts-build: concepts-inject concepts-emit
	mdbook build docs/concepts

## concepts-serve: preview concept catalog locally
concepts-serve: concepts-inject concepts-emit
	mdbook serve docs/concepts

## clean: hapus build artifacts
clean:
	rm -rf bin/ dist/ build/ docs/concepts/book/ coverage.txt coverage.html
	rm -f docs/concepts/src/concept-links.json

.PHONY: help install-tools migrate-up migrate-down migrate-status seed-demo reset-demo sqlc \
        build dev test coverage lint \
        web-install web-dev web-build web-test web-lint \
        concepts-inject concepts-emit concepts-lint concepts-build concepts-serve \
        clean
