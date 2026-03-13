# Backlog

## Now

- Auth session-first user hydration
  - Check session/store for current user before decoding JWT on every request.
  - Persist `CurrentUser` in session after successful authentication.
  - Preserve security guarantees and avoid stale user data.

## Done

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
