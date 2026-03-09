# Miconsul: patient appointment planner and notification center

[![Tests](https://github.com/edgarsilva/miconsul/actions/workflows/tests.yml/badge.svg)](https://github.com/edgarsilva/miconsul/actions/workflows/tests.yml)

Based on my GoScaffold project which allows you to quickly set-up Ready to
deploy Web application projects using Go, SQLite with GORM, Templ with HTMX
and DaisyUI/TailwindCSS:

-   Go Web Server: [Fiber](https://docs.gofiber.io/)
-   Database: [SQLite3 (or PostreSQL/MySQL)](https://sqlite.org/index.html)
-   ORM/SQL Query Builder: [GORM](https://gorm.io/docs/)
-   HTML/Templating: [Templ](https://templ.guide/)
-   Client/Server interactions: [HTMX](https://htmx.org/)
-   Client Interactivity: [AlpineJS](https://alpinejs.dev/start-here)
-   UI/CSS: [DaisyUI](https://daisyui.com)/[TailwindCSS](https://tailwindcss.com/)

## The MVP App

[vokoscreenNG-2024-07-23_17-00-32.webm](https://github.com/user-attachments/assets/f6915e3a-bb64-4a34-8ccc-78bec186f4a3)

## Getting Started

These instructions will get you a copy of the project up and running
on your local machine for development, scaffolding and testing purposes.

Other sections still pending in the README:

Note: _working_ but pending readme section

1. Automated production deployments with Coolify
2. Sqlite Litestream backups
3. Object storage with Minio (used by Litestream)
4. Development feature brances.

## Makefile

I've added `make` recipes/targets for the most common tasks, you can list them by
running `make`.

```bash
$ make

Meta
  help                        Show this help with available tasks

Setup
  install                     Installs deps 🥐 Bun, 🪿 goose, 🛕 templ and  go-localize
  install/go-localize         Install go-localize CLI

Code Quality
  fmt                         Run go fmt
  vet                         Run go vet (after fmt)
  lint                        Alias for vet

Frontend
  tailwind/build              Build Tailwind CSS
  tailwind/watch              Watch Tailwind CSS
  templ/build                 Generate Templ files (depends on tailwind)
  templ/watch                 Watch Templ
  locales/build               Build locales with go-localize
  locales/normalize           Remove volatile timestamp from generated locales file

Build & Run
  build                       Build Go binary with fts5
  start                       Start the built binary
  run                         Run via go run (generates Templ first)
  air/watch                   Run in dev mode with air (installs if missing)
  dev                         Start infra, then tailwind/watch, templ/watch, and air/watch

Tests
  test                        Run all tests
  test/race                   Run all tests with race detector
  test/v                      Verbose tests
  test/unit                   Run unit tests
  test/unit/c                 Run unit tests
  test/unit/v                 Run unit tests in verbose mode
  test/integration            Run integration tests
  test/coverage               Coverage

Database & Migrations
  db/create                   Create DB (migrations run by app/seed bootstrap)
  db/delete                   Delete DB (interactive confirmation)
  db/setup                    Recreate DB, apply migrations, and seed
  db/reset                    Alias for full DB reset (drop/migrate/seed)
  db/dump_schema              Dump DB schema
  migrations/apply            Apply migrations with goose
  db/migrate                  Alias for migrations/apply
  db/seed                     Run DB seeds (baseline + randomized bulk)
  migrations/status           Show migrations status
  migrations/rollback         Roll back last migration
  migrations/redo             Redo last migration

Docker
  docker/up                   Start docker infra services (without app)
  docker/dev                  Start all docker services including app
  docker/detached             docker compose up -d
  docker/down                 docker compose down
  docker/logs                 Follow logs for all docker services
  docker/app-logs             Follow app container logs
  docker/lgtm-logs            Follow LGTM stack logs
  docker/build                Rebuild the app image

Observability
  obs/load                    Run continuous synthetic traffic (~40 RPM total)
  obs/load/light              Run lighter synthetic traffic (~20 RPM total)
  obs/load/medium             Run medium synthetic traffic (~40 RPM total)
  obs/load/heavy              Run heavier synthetic traffic (~80 RPM total)
  load/test                   Run authenticated oha load test (30s, 30 concurrency)
```

### Install dependencies and tooling (not Go itself)

Install dev environment tooling, you must have `go` correctly installed and in
your path.

It will install:

-   bun: for TailwindCSS
-   TailwindCSS: plugins
-   goose: for migrations
-   templ: to build/compile templ files into go code

### Telemetry

The app uses OpenTelemetry tracer naming with dot notation by default:

-   `OTEL_SERVICE_NAME=miconsul`
-   `OTEL_TRACER_SERVER=miconsul.server`
-   `OTEL_TRACER_AUTH=miconsul.auth`

`OTEL_SERVICE_NAME` is used as the resource `service.name` and tracer names are
passed explicitly at bootstrap/service construction time.

For a practical local verification checklist (traffic generation, Grafana
queries, and units reference), see:

- `docs/observability-runbook.md`

### Testing

The test suite now includes a reusable integration harness and focused
regression coverage for high-risk routes.

- Default test DB mode is file-backed SQLite (stable local behavior).
- Optional in-memory mode can be enabled for faster ephemeral runs:

```bash
MICON_TEST_SQLITE_INMEMORY=1 go test ./...
```

`MICON` in `MICON_TEST_SQLITE_INMEMORY` is a short project prefix from
`miconsul`, used to avoid collisions with generic env var names.

For test strategy and harness details, see:

- `docs/testing.md`

Maintenance rule: handler/route bugfixes should include a regression test update in the matching `tests/<service>_handlers_test.go` suite.

CI quality gate (required on pull requests):

- `go test ./...`
- `go test -race ./...`

Quick commands:

```bash
# full suite
make test

# race detector pass
make test/race

# verbose tests
make test/v

# integration tests package
make test/integration

# full suite using in-memory sqlite harness mode
MICON_TEST_SQLITE_INMEMORY=1 make test
```

### DB - Seed Data

Create deterministic baseline data (including an admin user) and randomized bulk
records for local testing:

```bash
make db/seed
```

You can also run the seeder command directly and customize amounts:

```bash
go run -tags fts5 cmd/seed/main.go --users=2 --clinics=10 --patients=20 --appointments=40
```

To attach clinics/patients/appointments to a specific existing user:

```bash
go run -tags fts5 cmd/seed/main.go --owner-email="you@example.com" --users=0
```

If that user doesn't exist yet, add `--ensure-owner` to create it first.

```bash
make install
```

### DB - Create Database

Recreates the SQLite database, runs migrations, and applies seeds.

```bash
make db/setup
```

### Development

To run the app in dev mode, will auto generate css, templ
files and translations and enable auto reloading of the server on file changes
(not browser, just refresh the page, hit [f5] and done).

```bash
make dev
```

## Overall architecture guidelines for new features (WIP)

![image](https://github.com/edgarsilva/miconsul/assets/518231/6c270679-a3dc-432b-9394-08c7857eb1ea)

## Overall Data Models and ERD (WIP needs updates)

![image](https://github.com/edgarsilva/miconsul/assets/518231/c37e3599-65d6-4e73-814b-54aa91576b3b)
