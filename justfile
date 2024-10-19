set dotenv-load

# Display this list of recipes
default:
	just --list

# Install deps 🥐 Bun, 🪿 goose and 🛕 templ
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

# Run the Go formatter/linter
fmt:
	go fmt ./...

# Run go vet to detect code issues
vet: fmt
	go vet ./...

# Generate Tailwind styles
tailwind:
	@echo "🌬️ Generating Tailwind CSS styles..."
	~/.bun/bin/bun x tailwindcss -i ./styles/global.css -o ./public/global.css --minify

# Watch Tailwind styles
tailwind-watch:
	@echo "🌬️ Watching Tailwind CSS styles..."
	~/.bun/bin/bun x tailwindcss -i ./styles/global.css -o ./public/global.css --watch

# Generate Templ files
templ: tailwind
	@echo "🛕 Generating Templ files..."
	${GOPATH}/bin/templ generate

# Watch Templ files
templ-watch:
	@echo "🛕 Watching Templ files..."
	${GOPATH}/bin/templ generate --watch

# Build the app
build: templ
	@echo "📦 Building"
	@echo "🤖 go build..."
	go build -tags fts5 -o bin/app cmd/app/main.go

# Start the app using the build binary
start: migration-apply
	@echo "👟 Starting the app..."
	bin/app

# Run the app
run: templ vet
	@echo "👟 Running app..."
	@echo "🤖 go run..."
	go run -tags fts5 cmd/app/main.go

# Start the app in dev mode
dev:
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

# Run tests
test:
	@echo "Testing all"
	go test ./... -coverprofile=coverage/c.out

# Run unit-tests
unit-test:
	@echo "Testing unit"
	go test -v ./internal/... -coverprofile=coverage/unit_c.out

# Run integration-test
integration-test:
	@echo "Testing integration"
	go test -v ./tests/... -coverprofile=coverage/int_c.out

# Clean builds
clean:
	@echo "Cleaning builds..."
	rm bin/*

# Create Database
[group('db')]
db-create:
	touch database/app.sqlite
	just db-migrate

# Deletes the DB giving you a choice.
[group('db')]
db-delete:
	@read -p "Do you want to delete the DB (you'll loose all data)? [y/n] " choice; \
	if [ "$$choice" != "y" ] && [ "$$choice" != "Y" ]; then \
		echo "Exiting..."; \
		exit 1; \
	else \
		rm -f database/*.sqlite*; \
	fi; \

# Set up the DB by running delete, create and migrate
[group('db')]
db-setup:
	just db-create
	just db-migrate

# Dumps the DB schema to ./database/schema.sql
[group('db')]
db-dump-schema:
  sqlite3 database/app.sqlite '.schema' > ./database/schema.sql


# Migrates the DB to latest migration
[group('db')]
[group('migration')]
migration-apply:
	@echo "🪿 running migrations with goose before Start"
	${GOPATH}/bin/goose up

# Creates a new migration for the DB
[group('db')]
[group('migration')]
migration-create arg_name:
	${GOPATH}/bin/goose create {{arg_name}} sql

# Lists the DB migration status
[group('db')]
[group('migration')]
migration-status:
	${GOPATH}/bin/goose status

# Rollbacks last migration
[group('db')]
[group('migration')]
migration-rollback:
	${GOPATH}/bin/goose down

# Redos the last migration
[group('db')]
[group('migration')]
migration-redo:
	${GOPATH}/bin/goose redo

# Starts the docker services
[group('docker')]
docker-up:
	@echo " Docker services up"
	docker compose up

# Starts the docker services detached
[group('docker')]
docker-up-detached:
	@echo " Docker up detached"
	docker compose up -d

# Terminates the docker services
[group('docker')]
docker-down:
	@echo " Docker down"
	docker compose down

# Shows app service logs
[group('docker')]
docker-logs:
	@echo " Docker app logs "
	docker compose logs app -f

# Rebuild the docker image (for Dockerfile changes)
[group('docker')]
docker-build:
	docker compose up -d --no-deps --build app

