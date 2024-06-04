# Simple Makefile for a Go project

# Build the application
all: build

install/deps:
	@echo "📦 Installing OS deps"
	@echo "🤐 Installing unzip and tar"
	sudo apt-get install -y --no-install-recommends unzip tar
	make install

install:
	@echo "📦 Installing dependencies"
	@echo "🥐 Installing bun (for tailwindcss)"
	curl -fsSL https://bun.sh/install | bash
	@echo "🌬️ Installing TailwindCSS plugins"
	~/.bun/bin/bun install tailwindcss -d
	@echo "🛕 installing Templ"
	go install github.com/a-h/templ/cmd/templ@latest
	@echo "⚠️ ⚠️ ⚠️ Paths start⚠️ ⚠️ ⚠️ "
	ls -l
	ls -l bin
	@echo "⚠️ ⚠️ ⚠️ Paths end⚠️ ⚠️ ⚠️ "

build:
	@echo "📦 Building"
	@echo "🌬️ Generating Tailwind CSS styles..."
	~/.bun/bin/bunx tailwindcss -i ./styles/global.css -o ./public/global.css
	@echo "🛕 Generating Templ files..."
	./bin/templ generate
	@echo "🤖 go build..."
	go build -tags fts5 -o bin/app cmd/app/main.go

# Run the application
start:
	@echo "👟 Starting the app..."
	bin/app

# Run the application
run:
	@echo "👟 Running app..."
	@echo "🌬️ Generating Tailwind CSS styles..."
	~/.bun/bin/bunx tailwindcss -i ./styles/global.css -o ./public/global.css
	@echo "🛕 Generating Templ files..."
	templ generate
	@echo "🤖 go run..."
	go run cmd/app/main.go -tags fts5

# Test the application (integration)
test-integration:
	@echo "Testing..."
	go test ./tests -v

test-unit:
	@echo "Testing..."
	go test ./internal/...

# Clean the binary
clean:
	@echo "Cleaning..."
	rm bin/*

# Live Reload <- not hot reload on the browser
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

.PHONY: all build run test clean
