# Backlog

## Done

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
