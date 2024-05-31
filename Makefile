# Simple Makefile for a Go project

# Build the application
all: build

build:
	@echo "📦 Building"
	@echo "🌬️ Generating Tailwind CSS styles..."
	@bunx tailwindcss -i ./styles/global.css -o ./public/global.css
	@echo "🛕 Generating Templ files..."
	@templ generate
	@echo "🤖 go build..."
	@go build -tags fts5 -o bin/app cmd/app/main.go

# Run the application
run:
	@echo "👟 Running app..."
	@echo "🌬️ Generating Tailwind CSS styles..."
	@bunx tailwindcss -i ./styles/global.css -o ./public/global.css
	@echo "🛕 Generating Templ files..."
	@templ generate
	@echo "🤖 go run..."
	@go run cmd/app/main.go -tags fts5

# Test the application (integration)
test-integration:
	@echo "Testing..."
	@go test ./tests -v

test-unit:
	@echo "Testing..."
	@go test ./internal/...

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm bin/*

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
