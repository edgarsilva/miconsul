# syntax=docker/dockerfile:1.7

# ---------- GO BUILD ----------
FROM golang:1.26-bookworm AS build
WORKDIR /src

# Copy module files first to maximize cache hit rate.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download && go mod verify

# Copy source and build app binary (CGO required for SQLite/FTS5).
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -tags fts5 -o /out/miconsul ./cmd/app

# ---------- RUNTIME ----------
FROM debian:bookworm-slim AS runtime

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates tzdata curl wget unzip && \
    rm -rf /var/lib/apt/lists/*

RUN groupadd --gid 1000 miconsul && useradd --uid 1000 --gid miconsul --home /app --create-home miconsul

WORKDIR /app
RUN mkdir -p /app/bin /app/public /app/store && chown -R miconsul:miconsul /app

COPY --from=build /out/miconsul /app/bin/miconsul
COPY --chown=miconsul:miconsul public /app/public

USER miconsul

ENTRYPOINT ["/app/bin/miconsul"]
