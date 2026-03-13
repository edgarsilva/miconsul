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

## Later

- Templ toolchain alignment
  - Align `github.com/a-h/templ` module version with the generator used locally to avoid version skew warnings.
