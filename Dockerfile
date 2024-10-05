FROM golang:1.23

# Set destination for COPY
WORKDIR /app

RUN apt-get update && apt-get install -y unzip tar curl wget

# Install just utility
RUN curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin

# Install goose for DB migrations
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# Install bun (for TailwindCSS)
RUN curl -fsSL https://bun.sh/install | bash

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy app source code to /app inside container
COPY ./ .

# Install templ (Go HTML template generator)
RUN go install github.com/a-h/templ/cmd/templ@v0.2.778

# Build TailwindCSS plugins and the Go application
RUN ~/.bun/bin/bun install
RUN just build

# Start the application
CMD ["just", "start"]
