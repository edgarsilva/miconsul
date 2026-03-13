# Backlog

## Now

- Auth provider decoupling
  - Move provider-specific signin metadata (path, query/session keys, redirect behavior) behind provider/authenticator boundaries.
  - Keep generic auth handlers provider-agnostic.
  - Preserve current runtime behavior while introducing clearer seams.

## Next

- Auth file-splitting guardrail
  - Prefer cohesive `handlers.go` (transport) and `service.go` (business orchestration).
  - Split files only when a concern is clearly standalone and reused, to avoid fragmentation.
  - Keep structural refactors separate from behavior changes.

- Auth session-first user hydration
  - Check session/store for current user before decoding JWT on every request.
  - Persist `CurrentUser` in session after successful authentication.
  - Preserve security guarantees and avoid stale user data.

## Later

- Templ toolchain alignment
  - Align `github.com/a-h/templ` module version with the generator used locally to avoid version skew warnings.
