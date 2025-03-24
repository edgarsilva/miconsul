ifneq ($(wildcard ./.env),)
  include .env
  export
else
  env_check = $(shell echo "🟡 WARNING: .env file not found! continue only with exported shell env variables\n\n")
  $(info ${env_check})
  $(info )
endif

.DEFAULT: help
.PHONY: tailwind templ fmt vet buildset help

# Displays this list of tasks in the Makefile
help:
	@awk '/^[a-zA-Z0-9 _-\/]+:/ { if (prev_comment) { print $$1, "#", prev_comment } else { print $$1 }; prev_comment="" } { if (/^#/) { sub(/^# */, "", $$0); prev_comment=$$0 } }' $(MAKEFILE_LIST)

# Installs deps 🥐 Bun, 🪿 goose and 🛕 templ
install:
	@echo "📦 Installing dependencies"
	@echo "🥐 Installing bun (for tailwindcss)"
	curl -fsSL https://bun.sh/install | bash
	@echo "🌬️ Installing TailwindCSS plugins"
	~/.bun/bin/bun add -D tailwindcss
	~/.bun/bin/bun add -D daisyui@latest
	~/.bun/bin/bun add -D @tailwindcss/typography
	@echo "🪿 installing goose"
	go install github.com/pressly/goose/v3/cmd/goose@latest
	@echo "🛕 installing Templ"
	go install github.com/a-h/templ/cmd/templ@latest

# Runs the Go formatter/linter
fmt:
	go fmt ./...

# Runs go vet to detect code issues
vet: fmt
	go vet ./...

.PHONY: tailwind
tailwind:
	@echo "🌬️ Generating Tailwind CSS styles..."
	~/.bun/bin/bun x @tailwindcss/cli -i ./styles/global.css -o ./public/global.css --minify

.PHONY: tailwind/watch
tailwind/watch:
	@echo "🌬️ Watching for Tailwind CSS style changes..."
	~/.bun/bin/bun x @tailwindcss/cli -i ./styles/global.css -o ./public/global.css --minify --watch

templ: tailwind
	@echo "🛕 Generating Templ files..."
	${GOPATH}/bin/templ generate

.PHONY: templ/watch
templ/watch:
	@echo "🛕 Watching for Templ file changes..."
	${GOPATH}/bin/templ generate --watch -v

build: templ
	@echo "📦 Building"
	@echo "🤖 go build..."
	go build -tags fts5 -o bin/app cmd/app/main.go

# Start the app using the built binary
start: migrations/apply
	@echo "👟 Starting the app..."
	bin/app

# Runs the app using go run
run: templ
	@echo "👟 Running app..."
	@echo "🤖 go run..."
	go run -tags fts5 cmd/app/main.go

.PHONY: air/watch
# Starts the app in dev/watch mode
air/watch:
	@if command -v air > /dev/null; then \
	    air; \
	    echo "Running in dev mode and Watching files...";\
	else \
	    read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
	    if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
	        go install github.com/cosmtrek/air@latest; \
	        air; \
	        echo "Watching...";\
	    else \
	        echo "You chose not to install air. Exiting..."; \
	        exit 1; \
	    fi; \
	fi

# start all watch processes in parallel.
dev:
	make -j3 tailwind/watch templ/watch air/watch

# Run tests
test:
	@echo "Testing all"
	go test ./... -coverprofile=coverage/c.out

# Run unit-tests
.PHONY: test/unit
test/unit:
	@echo "Testing unit"
	go test -v ./internal/... -coverprofile=coverage/unit_c.out

# Run integration test
.PHONY: test/integration
test/integration:
	@echo "Testing integration"
	go test -v ./tests/... -coverprofile=coverage/int_c.out

# Deletes build file binaries
clean:
	@echo "Cleaning builds..."
	rm bin/*

.PHONY: db/create
db/create:
	touch database/app.sqlite
	make migrate

# Deletes the DB giving you a choice to opt out
db/delete:
	@read -p "Do you want to delete the DB (you'll loose all data)? [y/n] " choice; \
	if [ "$$choice" != "y" ] && [ "$$choice" != "Y" ]; then \
		echo "Exiting..."; \
		exit 1; \
	else \
		rm -f database/*.sqlite*; \
	fi; \

# Sets up the DB by running delete, create and migrate
db/setup:
	make db/delete
	make db/create
	make migrate

# Dumps the DB schema to ./database/schema.sql
db/dump_schema:
	sqlite3 database/app.sqlite '.schema' > ./database/schema.sql

# Runs the migrations
migrations/apply:
	@echo "🪿 running migrations with goose before Start"
	${GOPATH}/bin/goose up

# [Migrations]
# Runs the migrations (alias of migrations/apply)
migrate: migrations/apply

# [Migrations]
# Creates a migration file e.g. migrations/create migration_name
migrations/create arg_name:
	${GOPATH}/bin/goose create {{arg_name}} sql

# [Migrations]
migrations/status:
	${GOPATH}/bin/goose status

# [Migrations]
migrations/rollback:
	${GOPATH}/bin/goose down

# [Migrations]
migrations/redo:
	${GOPATH}/bin/goose redo

# Starts the docker-compose services
docker/up:
	@echo " Docker services up"
	docker compose up

# Starts the docker services detached
docker/detached:
	@echo " Docker up detached"
	docker compose up -d

# Terminates the docker services
docker/down:
	@echo " Docker down"
	docker compose down

# Shows app service logs
docker/logs:
	@echo " Docker app logs "
	docker compose logs app -f

# Rebuild the docker image (for Dockerfile changes)
docker/build:
	docker compose up --no-deps --build app
