# Miconsul

[![Tests](https://github.com/edgarsilva/miconsul/actions/workflows/tests.yml/badge.svg)](https://github.com/edgarsilva/miconsul/actions/workflows/tests.yml)

Miconsul is a patient appointment planner and notification center.

## Stack

- Go + Fiber v3
- SQLite + GORM
- Templ + HTMX + DaisyUI/TailwindCSS
- OpenTelemetry (traces, metrics, logs)

## Quick Start

Prerequisites:

- Go (with CGO support for SQLite)
- `make`
- Docker (optional, for local observability stack)

Install project tooling:

```bash
make install
```

Run in development mode (Tailwind + Templ watch + hot reload):

```bash
make dev
```

Run once without watchers:

```bash
make run
```

## Common Commands

List all available tasks:

```bash
make
```

Most used commands:

```bash
# quality
make fmt
make vet

# generate frontend artifacts
make templ/build
make locales/build

# tests
make test
make test/race
make test/coverage

# database
make db/setup
make db/seed
make migrations/status
```

## Testing

Default tests:

```bash
make test
```

Optional in-memory sqlite mode for faster ephemeral runs:

```bash
MICON_TEST_SQLITE_INMEMORY=1 go test ./...
```

Coverage helpers:

```bash
make test/coverage
make test/coverage/service-leaderboard
make test/coverage/html
```

More details: `docs/testing.md`.

## Database and Seeding

Recreate DB, run migrations, and seed:

```bash
make db/setup
```

Run seeds only:

```bash
make db/seed
```

Seed command with custom amounts:

```bash
go run -tags fts5 cmd/seed/main.go --users=2 --clinics=10 --patients=20 --appointments=40
```

## Observability

Local load generation helpers:

```bash
make obs/load/light
make obs/load/medium
make obs/load/heavy
```

Runbook: `docs/observability-runbook.md`.

## Architecture

Project entry points:

- App: `cmd/app/main.go`
- Seed command: `cmd/seed/main.go`
- Router wiring: `internal/routes/router.go`
- HTTP server/middleware: `internal/server`
- Services/handlers: `internal/service`
- Views: `internal/view/*.templ`

Architecture guidelines and diagrams:

- `docs/architecture.md`

## Deployment

Deployment docs are being prepared:

- `docs/deployment.md`

## Maintenance Plan

Current cleanup/refactor stream:

1. Refresh README and remove stale sections.
2. Remove legacy `internal/lib/url` usage and rely on bootstrap env wiring.
3. Remove untyped locals in appointments/dashboard view data flow.
4. Reuse DB logger path for migrations so migration logs reach Loki/OTEL.

## Media

MVP demo:

- [vokoscreenNG-2024-07-23_17-00-32.webm](https://github.com/user-attachments/assets/f6915e3a-bb64-4a34-8ccc-78bec186f4a3)
