# syntax=docker/dockerfile:1.7

# ---------- GO BUILD ----------
FROM golang:1.27-bookworm AS build
WORKDIR /src

# Install the official Node.js binary after verifying its published checksum.
ARG NODE_VERSION=24.18.0
ARG NODE_SHA256=783130984963db7ba9cbd01089eaf2c2efb055c7c1693c943174b967b3050cb8
RUN curl -fsSLo /tmp/node.tar.gz https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-x64.tar.gz \
    && echo "${NODE_SHA256}  /tmp/node.tar.gz" | sha256sum -c - \
    && tar -xzf /tmp/node.tar.gz -C /usr/local --strip-components=1 --no-same-owner \
    && rm /tmp/node.tar.gz

# Copy module files first to maximize cache hit rate.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download && go mod verify

# Install locked frontend dependencies before copying application sources.
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci --ignore-scripts --allow-remote=all --no-audit --no-fund --loglevel=error

# Copy source, generate CSS, and build the app binary.
COPY . .
RUN npm run css:build
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
COPY --chown=miconsul:miconsul --from=build /src/public /app/public

USER miconsul

ENTRYPOINT ["/app/bin/miconsul"]
