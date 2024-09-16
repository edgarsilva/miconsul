ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.PHONY: all clean

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

build:
	@echo "📦 Building"p
	@echo "🌬️ Generating Tailwind CSS styles..."
	~/.bun/bin/bunx tailwindcss -i ./styles/global.css -o ./public/global.css --minify
	@echo "🛕 Generating Templ files..."
	${GOPATH}/bin/templ generate
	@echo "🤖 go build..."
	go build -tags fts5 -o bin/app cmd/app/main.go

start:
	make db/migrate
	@echo "👟 Starting the app..."
	bin/app

run:
	@echo "👟 Running app..."
	@echo "🌬️ Generating Tailwind CSS styles..."
	~/.bun/bin/bunx tailwindcss -i ./styles/global.css -o ./public/global.css --minify
	@echo "🛕 Generating Templ files..."
	${GOPATH}/bin/templ generate
	@echo "🤖 go run..."
	go run -tags fts5 cmd/app/main.go

# Local development with Live Reload <- not hot reload on the browser only of the server
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

test:
	@echo "Testing all"
	go test ./... -coverprofile=coverage/c.out


test/unit:
	@echo "Testing unit"
	go test -v ./internal/... -coverprofile=coverage/unit_c.out

test/integration:
	@echo "Testing integration"
	go test -v ./tests/... -coverprofile=coverage/int_c.out

clean:
	@echo "Cleaning builds..."
	rm bin/*

db/create:
	touch database/app.sqlite
	make db/migrate

db/delete:
	@read -p "Do you want to delete the DB (you'll loose all data)? [y/n] " choice; \
	if [ "$$choice" != "y" ] && [ "$$choice" != "Y" ]; then \
		echo "Exiting..."; \
		exit 1; \
	else \
		rm -f database/*.sqlite*; \
	fi; \

db/setup:
	make db/delete
	make db/create
	make db/migrate

db/dump-schema:
	sqlite3 database/app.sqlite '.schema' > ./database/schema.sql

db/migration:
	${GOPATH}/bin/goose create ${name} sql

db/status:
	${GOPATH}/bin/goose status

db/migrate:
	@echo "🪿 running migrations with goose before Start"
	${GOPATH}/bin/goose up

db/rollback:
	${GOPATH}/bin/goose down

db/redo:
	${GOPATH}/bin/goose redo

docker/build:
	docker build . -t go-containerized:latest

docker/run:
	docker run -e PORT=3000 -p 3000:3000 --name miconsul-app go-containerized:latest

docker/stop:
	docker stop miconsul-app

docker/clean:
	docker container rm miconsul-app

