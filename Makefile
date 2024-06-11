ifneq (,$(wildcard ./.env))
    include .env
    export
endif

all: install build

docker/build:
	docker build . -t go-containerized:latest

docker/run:
	docker run -e PORT=3000 -p 3000:3000 --name miconsul-app go-containerized:latest

docker/start:
	docker start miconsul-app

docker/stop:
	docker stop miconsul-app

docker/clean:
	docker container rm miconsul-app

install:
	@echo "📦 Installing dependencies"
	@echo "🥐 Installing bun (for tailwindcss)"
	curl -fsSL https://bun.sh/install | bash
	@echo "🌬️ Installing TailwindCSS plugins"
	~/.bun/bin/bun add -D tailwindcss
	~/.bun/bin/bun add -D daisyui@latest
	~/.bun/bin/bun add -D @tailwindcss/typography
	@echo "🛕 installing Templ"
	go install github.com/a-h/templ/cmd/templ@latest

build:
	@echo "📦 Building"
	@echo "🌬️ Generating Tailwind CSS styles..."
	~/.bun/bin/bunx tailwindcss -i ./styles/global.css -o ./public/global.css --minify
	@echo "🛕 Generating Templ files..."
	${GOPATH}/bin/templ generate
	@echo "🤖 go build..."
	go build -tags fts5 -o bin/app main.go

start:
	@echo "👟 Starting the app..."
	bin/app

run:
	@echo "👟 Running app..."
	@echo "🌬️ Generating Tailwind CSS styles..."
	~/.bun/bin/bunx tailwindcss -i ./styles/global.css -o ./public/global.css --minify
	@echo "🛕 Generating Templ files..."
	${GOPATH}/bin/templ generate
	@echo "🤖 go run..."
	go run main.go -tags fts5

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
	@echo "Testing unit"
	go test ./internal/...
	@echo "Testing integration"
	go test ./tests -v

clean:
	@echo "Cleaning..."
	rm bin/*

db/create:
	touch database/app.sqlite

db/reset:
	@read -p "Do you want to reset the DB (you'll loose all data)? [y/n] " choice; \
	if [ "$$choice" != "y" ] && [ "$$choice" != "Y" ]; then \
		echo "Exiting..."; \
		exit 1; \
	else \
		rm database/*.sqlite*; \
	fi; \

db/dump-schema:
	sqlite3 database/app.sqlite '.schema' > ./database/schema.sql

db/setup:
	db/reset
	db/create
	migrate/up

migrate/up:
	${GOPATH}/bin/goose up

migrate/down:
	${GOPATH}/bin/goose down

migrate/status:
	${GOPATH}/bin/goose status

migrate/redo:
	${GOPATH}/bin/goose redo

.PHONY: all install build start run test clean dev
