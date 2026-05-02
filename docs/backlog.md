# Backlog

## Next Up

- [external/runtime] Uptime Kuma monitor setup (environment ready)
  - Configure monitors in Kuma for `/livez`, `/readyz`, and optional `/startupz`.
  - Apply final interval/timeout/retry settings directly in the Kuma UI.
- [external/runtime] Uptime Kuma notifications and routing
  - Configure Slack/Discord/Email channels and label-based alert routing.
- [external/runtime] SLO-style alerts in Kuma/Grafana
  - Availability alert: `/livez` success rate over 5m.
  - Service health alert: `/readyz` failure streak threshold (for example 3 consecutive failures).
  - Degradation alert: `/readyz` latency above threshold.

- [feature/storage] Object storage uploads for images (RustFS S3-compatible)
  - Define storage abstraction and configuration wiring for local disk vs S3-compatible backends.
  - Upload and retrieval path for patient/profile images via RustFS.
  - Migration/fallback strategy for existing disk-backed image files.

- [feature/notifications] Multi-channel notifications baseline
  - Keep email templates/actions synced with appointment + professional details.
  - Add Telegram provider integration (credentials, send path, retry/error handling).
  - Add WhatsApp provider integration.
  - Add Facebook Messenger provider integration.
  - Define per-channel opt-in/opt-out + fallback policy (email as default fallback).

## Icebox

- [external/runtime] Beta tester release prep
  - Define exit criteria and communication checklist for first beta group.

- [infra/sessions] Optional Valkey-backed HTTP sessions (lowest priority)
  - Replace SQLite session storage with Valkey storage behind Fiber session middleware.
  - Keep fallback behavior and rollout checklist for local/dev environments.

## Done

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
