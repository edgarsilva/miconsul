# ---------- GO BUILD ----------
FROM golang:1.24-bookworm AS build
WORKDIR /app

# Enable module and build caching with BuildKit
RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	echo "cache warmup"

# Copy go module files first for caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy the rest of the source
COPY . .

# Build the binary (CGO on for SQLite)
ENV CGO_ENABLED=1
RUN go build -tags fts5 -o bin/app cmd/app/main.go

# ---------- RUNTIME ----------
FROM debian:bookworm-slim AS runtime

RUN apt-get update && \
	apt-get install -y sqlite3 --no-install-recommends && \
	apt-get install -y curl --no-install-recommends && \
	rm -rf /var/lib/apt/lists/*



# Create non-root user (already has nonroot:nonroot 65532)
RUN groupadd -g 1000 miconsul && useradd -u 1000 -m -g miconsul miconsul
USER miconsul
WORKDIR /app
RUN mkdir /app/store

# App binary and runtime assets
COPY . .
COPY --from=build /app/bin/app /app/bin/miconsul

# Start the application
ENTRYPOINT ["/app/bin/miconsul"]
