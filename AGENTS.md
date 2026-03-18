# AGENTS

This file provides repo-specific guidance for coding assistants working on this project.

## Project Snapshot

- Stack: Go, Fiber v3, GORM, SQLite, templ, HTMX, Tailwind/DaisyUI.
- App entrypoint: `cmd/app/main.go`.
- HTTP server and middleware: `internal/server/`.
- Routes: `internal/routes/router.go`.
- Services/handlers: `internal/services/**`.
- templ views: `internal/views/*.templ` (generated files are `*_templ.go`).
- Email templates: `internal/mailer/*.templ`.

## High-Priority Rules

- Do not edit generated templ files (`*_templ.go`) by hand.
- Edit `.templ` sources, then regenerate.
- Keep Fiber APIs on v3 conventions (already migrated):
  - `fiber.Ctx` (not `*fiber.Ctx`)
  - `c.Redirect().To(...)` and `c.Redirect().Status(...).To(...)`
  - `c.Bind().Body(...)` instead of legacy body parser helpers
- Prefer `c.Context()` when passing request context to non-Fiber libraries (DB, tracing, etc.).
- In service/handler files, keep primary exported handlers/entrypoints near the top and move private helpers/utilities to the bottom.
- Avoid code fragmentation: keep module logic cohesive in `handlers.go` (transport) and `service.go` (business orchestration); split only when a concern is clearly standalone and reused.
- Keep structural refactors separate from behavior changes.
- Avoid introducing runtime `os.Getenv` reads in application/service code; route configuration access through `internal/lib/appenv` instead. If you encounter new `os.Getenv` usage outside mailer or env-loading boundaries, warn and offer to move it into `appenv.Env`.
- Request locals boundary:
  - `c.Locals(...)` writes are allowed in middleware auth/session identity binding.
  - `c.Locals(...)` reads are allowed only in `internal/views/ctx.go` and `internal/server/request.go`.
  - Everywhere else should use `s.CurrentUser(c)` and view context APIs instead of reading locals directly.

## templ Guidance Source

- For templ syntax/rules, use: `docs/ai/templ_llms_compact.md`.
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

## Git Commit Preference

- Never create a git commit unless the user explicitly asks in that moment.
- After code changes, always pause and ask whether to commit.
- Always let the user review changes before committing; do not commit immediately after implementation unless the user explicitly requests commit at that point.

## Branch Workflow

- Start every feature/fix/refactor on a new branch created from a freshly synced `main`.
- Avoid continuing development directly on `main`; keep `main` aligned with `origin/main` between workstreams.
- Never commit directly on `main`.
- If work starts while on `main`, create a feature branch before making changes or creating commits.

## Pull Request Workflow

- Never merge a pull request on behalf of the user unless they explicitly ask for merge in that moment.
- Default behavior after opening a PR: share the PR URL, wait for user review, and let the user merge manually.
