# Backlog

## Next Up

- [feature/dashboard] Populate FeedEvent on appointment changes and wire Dashboard Feed widget
  - [ ] Create FeedEvent records on appointment create/update/cancel/complete.
  - [ ] Update Dashboard Feed widget to read from FeedEvent table instead of static/dummy data.
  - [ ] Add FeedEvent query scoped to current user + recent time window.

- [feature/notifications] Multi-channel notifications baseline
  - Keep email templates/actions synced with appointment + professional details.
  - Add Telegram provider integration (credentials, send path, retry/error handling).
  - Add WhatsApp provider integration.
  - Add Facebook Messenger provider integration.
  - Define per-channel opt-in/opt-out + fallback policy (email as default fallback).

- [feature/search] Global Ctrl+K search modal
  - [ ] Add keyboard shortcut (`Ctrl+K`) to open a global search modal.
  - [ ] Search across appointments, clinics, and patients from a single input.
  - [ ] Support keyboard navigation + enter-to-open selected result.
  - [ ] Keep existing index search endpoints; add a dedicated global search endpoint contract.
  - [ ] Define result grouping and ranking order (appointments first vs mixed relevance).

- [feature/htmx4] Replace OOB price update with partials pattern
  - [ ] Refactor appointment clinic-search response to use HTMX 4 partial swap flow instead of `hx-swap-oob`.
  - [ ] Keep clinic list and price updates independent targets with explicit swap contracts.
  - [ ] Add regression check for clinic search + price update on repeated searches and back navigation.

- [feature/storage] Object storage uploads for images (RustFS S3-compatible)
  - Define storage abstraction and configuration wiring for local disk vs S3-compatible backends.
  - Upload and retrieval path for patient/profile images via RustFS.
  - Migration/fallback strategy for existing disk-backed image files.

- [infra/build] Generate Tailwind CSS at Docker build time
  - [ ] Add Bun/Node build stage in `Dockerfile` to compile `styles/global.css` → `public/global.css`.
  - [ ] Stop committing generated `public/global.css`; treat it as a build artifact.
  - [ ] Keep runtime image free of Bun/Node binaries.

- [infra/build] Define templ generation policy for CI/image builds
  - [ ] Decide and document templ artifact strategy (`*.templ` source vs committed generated `*_templ.go`).
  - [ ] Enforce one canonical source of truth via CI if generating at build time.

## Icebox

- [external/runtime] Beta tester release prep
  - Define exit criteria and communication checklist for first beta group.

- [feature/feed-audit] Granular change tracking for FeedEvents (icebox)
  - Add `Attribute`, `From`, `To` fields to `FeedEvent` for audit-trail granularity.
  - Use case: "Patient appointment **price** changed **from** $50 **to** $75".
  - Migration: `ALTER TABLE feed_events ADD COLUMN attribute TEXT, "from" TEXT, "to" TEXT`.
  - Note: Previous unexported `onAttr`, `from`, `to` fields existed in Go struct but were ignored by GORM and never stored; removed in 2026-05-11 cleanup.

- [infra/sessions] Optional Valkey-backed HTTP sessions (lowest priority)
  - Replace SQLite session storage with Valkey storage behind Fiber session middleware.
  - Keep fallback behavior and rollout checklist for local/dev environments.

## Done

- Appointment index search parity with clinics/patients
  - Added `GET /appointments/search` HTMX index search endpoint.
  - Added appointments page search input that preserves active `timeframe`, `patientId`, and `clinicId` filters.
  - Extended appointment query filtering by patient/clinic identity fields while keeping existing filter behavior.

- Auth coverage hardening after migration checklist closure
  - Added explicit callback verification failure coverage for `GET /logto/callback` (`logto_error=callback`).
  - Added explicit `logto_skip_redirect` session key clearing branch coverage.
  - Added auth snapshot hydration TTL boundary coverage.
  - Added compact auth route contract + provider callback error matrix to `docs/testing.md`.

- Testing docs drift cleanup for auth migration status
  - Updated migration checklist in `docs/testing.md` to mark `auth` as done.
  - Captured remaining auth work as hardening tasks instead of migration status debt.

- Devx/docs toolchain manager guidance in README
  - Documented preferred local setup path (`mise`) while keeping alternatives valid (`asdf`, `homebrew`).
  - Clarified `make install/deps` (project deps) versus `make install/tools` (optional local CLIs).

- Devx setup target and toolchain ownership cleanup
  - Renamed setup flow to `make install/deps` and kept `make install` as alias.
  - Removed Bun toolchain installation from default Make setup path.
  - Added `check/bun` fail-fast message for missing Bun binary.
  - Split optional CLI installation under `make install/tools`.

- Observability runbook troubleshooting flow
  - Added response playbook for: `/readyz` failing while `/livez` is passing.

- Auth session-first user hydration
  - Added session-first auth snapshot hydration before JWT + DB fallback.
  - Persisted request identity snapshot after successful authentication.
  - Kept boundaries explicit: auth resolves identity, middleware binds locals, CurrentUser reads locals only.
  - Hardened session snapshot by storing token digest (SHA-256), not raw JWT.

- Auth provider decoupling
  - Moved provider signin metadata behind `Authenticator.Metadata()`.
  - Kept generic signin handler flow provider-agnostic.
  - Preserved behavior for current Logto setup.

- Auth file-splitting guardrail
  - Codified in `AGENTS.md` to keep `handlers.go` + `service.go` cohesive.
  - Explicitly requires splitting only for clearly standalone/reused concerns.
  - Reinforced separation of structural refactors from behavior changes.

- Templ toolchain alignment
  - Aligned `github.com/a-h/templ` module dependency with the generator version.
  - Verified with `go test ./...`.

- Appointment index search follow-ups
  - FTS5 `global_fts` table active for appointments/clinics/patients.
  - Native query performance sufficient; no meilisearch path needed yet.

- Uptime Kuma monitors
  - Endpoints `/livez`, `/readyz`, `/startupz` exposed and documented.
  - Monitors configured externally in Kuma.

- Logto tenant provisioning + Coolify deployment docs
  - Logto tenant provisioned; OAuth/Google identity sign-in working.
  - `docs/deployment.md` covers Coolify env vars, healthcheck expectations, and Logto wiring checklist.

- Production bootstrap guardrails (partial)
  - `COOKIE_SECRET` validated for 16/24/32 bytes at startup.
  - Post-migration admin auto-creation from `ADMIN_USER` + `ADMIN_PASSWORD`.
  - Mailers consume `appenv.Env` directly; no `os.Getenv` reads in mailer code.

- [feature/auth] Harden local signup confirm-email delivery
  - Signup confirm email sent synchronously; SMTP failures surface to user.
  - Env key alignment audited (`EMAIL_SENDER`, `EMAIL_FROM_ADDRESS`).
  - Raw goroutines replaced with `SendToWorker` for resend/reset flows.

- [infra/runtime] Add panic recovery to SendToWorker
  - `SendToWorker` wraps jobs with `recover()` + OTel span error recording.
  - `appointment/alerts.go` call sites updated to pass context.
  - Instrumentation methods extracted to `internal/server/instrumentation.go`.

- [feature/ui] Add relative time formatting to appointments and users lists
  - Added `github.com/dustin/go-humanize` dependency.
  - Created `RelativeTime()` helper with abbreviated format ("5m", "2h", "3d", "1y", "in 5h").
  - Added `ContextualTime()` for appointments: "Today @ 3:00pm", "Wednesday @ 11:30am", "Next Week", absolute dates for past.
  - Added `CmpContextualTime` templ component with timezone-aware tooltip.
  - Updated appointments index: contextual time display + color coding (today=accent, future=info, past=dimmed).
  - Updated users index: `UpdatedAt` shows relative time.
  - Added status filter buttons (confirmed/pending/rescheduled) with toggle behavior.
