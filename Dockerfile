FROM golang:1.22

# Set destination for COPY
WORKDIR /app

RUN apt-get update
RUN apt-get install -y unzip tar curl wget
RUN curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin

RUN echo "🪿 installing goose"
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

RUN echo "🥐 Installing bun (for tailwindcss)"
RUN curl -fsSL https://bun.sh/install | bash

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY ./ .

RUN echo "🛕 installing Templ"
RUN go install github.com/a-h/templ/cmd/templ@v0.2.778

# Build
RUN echo "🌬️ Installing TailwindCSS plugins"
RUN ~/.bun/bin/bun install
RUN just build

# Start
CMD ["just", "start"]

