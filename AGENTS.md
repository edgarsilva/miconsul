# AGENTS

This file provides repo-specific guidance for coding assistants working on this project.

## Project Snapshot

- Stack: Go, Fiber v3, GORM, SQLite, templ, HTMX, Tailwind/DaisyUI.
- App entrypoint: `cmd/app/main.go`.
- HTTP server and middleware: `internal/server/`.
- Routes: `internal/routes/router.go`.
- Services/handlers: `internal/service/**`.
- templ views: `internal/view/*.templ` (generated files are `*_templ.go`).
- Email templates: `internal/mailer/*.templ`.

## High-Priority Rules

- Do not edit generated templ files (`*_templ.go`) by hand.
- Edit `.templ` sources, then regenerate.
- Keep Fiber APIs on v3 conventions (already migrated):
  - `fiber.Ctx` (not `*fiber.Ctx`)
  - `c.Redirect().To(...)` and `c.Redirect().Status(...).To(...)`
  - `c.Bind().Body(...)` instead of legacy body parser helpers
- Prefer `c.Context()` when passing request context to non-Fiber libraries (DB, tracing, etc.).

## templ Guidance Source

- For templ syntax/rules, use: `docs/ai/templ-llms.compact.md`.
- Only load that document when the task touches templ/view/email templates.
- If task is not templ-related, skip it to keep context focused.

## Build/Test Workflow

- Regenerate templ output after templ changes:
  - `make templ` (preferred)
  - or `templ generate` if using the CLI directly
- Validate with:
  - `go test ./...`

## Change Scope Discipline

- Keep changes narrowly scoped to the user request.
- Avoid broad refactors when doing dependency upgrades.
- For dependency updates, report changed files and test status before committing.
