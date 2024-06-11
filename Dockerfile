FROM golang:1.22

# Set destination for COPY
WORKDIR /app

RUN apt-get update
RUN apt-get install -y unzip tar curl wget

RUN echo "🪿 installing goose"
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

RUN echo "🛕 installing Templ"
RUN go install github.com/a-h/templ/cmd/templ@latest

RUN echo "🥐 Installing bun (for tailwindcss)"
RUN curl -fsSL https://bun.sh/install | bash

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY ./ .

RUN echo "🌬️ Installing TailwindCSS plugins"
RUN ~/.bun/bin/bun install

# Run migrations
RUN echo "🪿 running migrations with goose"
RUN echo "GOOSE_DRIVER"
RUN echo ${GOOSE_DRIVER}
RUN echo "GOOSE_DBSTRING"
RUN echo ${GOOSE_DBSTRING}
RUN echo "GOOSE_MIGRATION_DIR"
RUN echo ${GOOSE_MIGRATION_DIR}
RUN ls -lh ${GOOSE_DBSTRING}
RUN make migrate/up

# Build
RUN make build

# Start
CMD ["make", "start"]

