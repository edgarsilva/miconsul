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

- Go 1.26+ (with CGO support for SQLite)
- `make`
- Bun
- Docker (optional, for local observability stack)

### Toolchain Manager Guidance

Preferred local toolchain path is `mise`.

Alternatives remain valid:

- `asdf`
- `homebrew`

Use your manager to install runtime/toolchain binaries (`go`, `bun`).

Then use project tasks for repository setup:

- `make install/deps`: installs project dependencies (`go mod download`, `bun install`)
- `make install/tools`: installs optional local CLIs (`templ`, `go-localize`)

### Environment Setup

Create your local environment file:

```bash
cp .env.example .env
```

Minimum app variables to verify/update in `.env`:

- `APP_ENV`, `APP_PORT`, `APP_NAME`, `APP_PROTOCOL`, `APP_DOMAIN`, `APP_VERSION`
- `COOKIE_SECRET`, `JWT_SECRET`
- `DB_PATH`, `SESSION_DB_PATH`

Optional but commonly used:

- `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_EXPORTER_OTLP_INSECURE`
- `JOBS_ENABLED`, `JOBS_UI_ENABLED`, `VALKEY_HOST`, `VALKEY_PORT`, `VALKEY_DB`
- `LOGTO_URL`, `LOGTO_APP_ID`, `LOGTO_APP_SECRET`, `LOGTO_RESOURCE`

Install project tooling:

```bash
make install/deps
```

Optional local CLI tools (`templ`, `go-localize`):

```bash
make install/tools
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

Runbook: `docs/observability_runbook.md`.

Jobs operations runbook: `docs/jobs_runbook.md`.

## Architecture

Project entry points:

- App: `cmd/app/main.go`
- Seed command: `cmd/seed/main.go`
- Router wiring: `internal/routes/router.go`
- HTTP server/middleware: `internal/server`
- Services/handlers: `internal/services`
- Views: `internal/views/*.templ`

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
