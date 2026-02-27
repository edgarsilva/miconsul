ifneq ($(wildcard ./.env),)
  include .env
  export
else
  env_check = $(shell echo "🟡 WARNING: .env file not found! continue only with exported shell env variables\n\n")
  $(info ${env_check})
endif

.DEFAULT_GOAL := help
SHELL := /bin/bash
GOBIN ?= $(shell go env GOBIN)

##@ Meta
help: ## Show this help with available tasks
	@awk 'BEGIN {FS = ":.*## "}; \
	/^[a-zA-Z0-9_\/-]+:.*## / { printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2 } \
	/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0,5) }' $(MAKEFILE_LIST)

##@ Setup
install: ## Installs deps 🥐 Bun, 🪿 goose and 🛕 templ
	@echo "📦 Installing dependencies"
	@echo "🥐 Installing bun (for tailwindcss)"
	curl -fsSL https://bun.sh/install | bash
	@echo "🌬️ Installing TailwindCSS plugins"
	bun add -D tailwindcss@latest
	bun add -D daisyui@latest
	bun add -D @tailwindcss/typography
	@echo "🪿 installing goose"
	go install github.com/pressly/goose/v3/cmd/goose@latest
	@echo "🛕 installing Templ"
	go install github.com/a-h/templ/cmd/templ@latest

##@ Code Quality
fmt: ## Run go fmt
	go fmt ./...

vet: fmt ## Run go vet (after fmt)
	go vet ./...

lint: vet ## Alias for vet

##@ Frontend
tailwind/build: ## Build Tailwind CSS
	@echo "🌬️ Generating Tailwind CSS styles..."
	bun x @tailwindcss/cli -i ./styles/global.css -o ./public/global.css --minify

tailwind/watch: ## Watch Tailwind CSS
	@echo "🌬️ Watching for Tailwind CSS changes..."
	bun x @tailwindcss/cli -i ./styles/global.css -o ./public/global.css --minify --watch

templ/build: tailwind/build ## Generate Templ files (depends on tailwind)
	@echo "🛕 Generating Templ files..."
	templ generate

templ/watch: ## Watch Templ
	@echo "🛕 Watching for Templ changes..."
	templ generate --watch -v

locales/build: ## Build locales with go-localize
	@echo "  Building locales"
	go-localize -input locales -output internal/lib/localize

##@ AI Docs
ai/templ-sync: ## Refresh templ LLM upstream snapshot and metadata
	@echo "🤖 Refreshing templ LLM docs snapshot..."
	./scripts/update-templ-llms.sh

##@ Build & Run
build: templ locales/build ## Build Go binary with fts5
	@echo "📦 Building"
	go build -tags fts5 -o bin/app cmd/app/main.go

start: ## Start the built binary
	@echo "👟 Starting the app..."
	bin/app

run: templ ## Run via go run (generates Templ first)
	@echo "👟 Running app..."
	go run -tags fts5 cmd/app/main.go

air/watch: ## Run in dev mode with air (installs if missing)
	@if command -v air > /dev/null; then \
	    air; \
	else \
	    read -p "Install air? [Y/n] " choice; \
	    if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
	        go install github.com/cosmtrek/air@latest; \
	        air; \
	    else \
	        echo "You chose not to install air. Exiting..."; \
	        exit 1; \
	    fi; \
	fi

dev: ## Start tailwind/watch, templ/watch, and air/watch in parallel
	$(MAKE) -j3 tailwind/watch templ/watch air/watch

##@ Tests
test: ## Run all tests with coverage
	@echo "Testing all"
	go test -race ./internal/...
	go test ./..

test/v: ## Verbose tests
	@echo "Testing all verbose"
	go test ./... -race -v

test/unit: ## Run unit tests
	@echo "Testing unit"
	go test -race ./internal/...

test/unit/c: ## Run unit tests
	@echo "Testing unit"
	go test ./internal/... -race -coverprofile=coverage/unit_c.out && go tool cover -func=coverage/unit_c.out

test/unit/v: ## Run unit tests in verbose mode
	@echo "Testing units --verbose"
	go test -v -race ./internal/...

test/integration: ## Run integration tests
	@echo "Testing integration"
	go test -v ./tests/... -coverprofile=coverage/int_c.out

test/coverage: ## Coverage
	go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage/c.out

##@ Test Coverage
cover:
	go test ./internal/... -covermode=atomic -coverpkg=./internal/... -coverprofile=coverage/c.out
	go tool cover -func=coverage/c.out

cover/html: cover
	go tool cover -html=coverage/c.out -o coverage/c.html
	@echo "Open coverage.html in your browser"
	google-chrome coverage/c.html

cover/missing: cover
	@awk -F '[: ,]+' 'NR>1 && $$NF==0 {printf "%s:%s-%s\n",$$1,$$2,$$4}' coverage.out | sort

##@ Cleanup
clean: ## Remove build artifacts
	@echo "Cleaning builds..."
	rm -f bin/*

##@ Database & Migrations
db/create: ## Create DB (and run migrations)
	$(MAKE) db/migrate

db/delete: ## Delete DB (interactive confirmation)
	@read -p "Delete DB (this is destructive)? [y/N] " choice; \
	if [ "$$choice" != "y" ] && [ "$$choice" != "Y" ]; then \
		echo "Exiting..."; exit 1; \
	else echo "deleting..."; fi

db/setup: ## Recreate DB and apply migrations
	$(MAKE) db/delete
	$(MAKE) db/create

db/dump_schema: ## Dump DB schema
	sqlite3 store/app.sqlite3 .schema > goose/schema.sql

migrations/apply: ## Apply migrations with goose
	@echo "🪿 running migrations with momma goose"
	$(GOBIN)/goose up

db/migrate: migrations/apply ## Alias for migrations/apply

.PHONY: migrations/create/%
migrations/create/%: ## Create a migration file: make migrations/create/add_column_to_table
	${GOBIN}/goose create $* sql

migrations/status: ## Show migrations status
	$(GOBIN)/goose status

migrations/rollback: ## Roll back last migration
	$(GOBIN)/goose down

migrations/redo: ## Redo last migration
	$(GOBIN)/goose redo

##@ Docker
docker/up: ## docker compose up (foreground)
	@echo " Docker services up"
	docker compose up

docker/detached: ## docker compose up -d
	@echo " Docker up detached"
	docker compose up -d

docker/down: ## docker compose down
	@echo " Docker down"
	docker compose down

docker/logs: ## Follow app logs
	@echo " Docker app logs "
	docker compose logs app -f

docker/build: ## Rebuild the app image
	docker compose up --no-deps --build app

.PHONY: help install fmt vet lint \
	tailwind tailwind/watch templ templ/watch locales/build \
	ai/templ-sync \
	build start run air/watch dev \
	test test/unit test/integration clean \
	db/create db/delete db/setup db/dump_schema \
	migrations/apply migrate migrations/create migrations/status migrations/rollback migrations/redo \
	docker/up docker/detached docker/down docker/logs docker/build
