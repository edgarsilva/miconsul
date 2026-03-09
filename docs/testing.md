# Testing Strategy

This project uses a layered testing approach focused on regression prevention,
route behavior confidence, and safe iterative refactors.

## Goals

- Catch high-risk route regressions early.
- Validate persistence behavior, not only HTTP responses.
- Keep tests readable and maintainable as the app grows.

## Current Baseline

- Reusable integration harness: `tests/helpers_test.go`
- Route regression coverage: `tests/regression_routes_test.go`

Covered regressions currently include:

- `/profile` POST update persistence
- patch/delete routes for appointment, patient, and clinic
- `/api/users` auth gating behavior
- `/appointments/search/clinics` route behavior

## Harness Design

`newTestHarness(t)` boots a real app stack for route-level tests:

- starts `server.Server` with registered routes
- creates isolated test DB and auto-migrates schema
- supports fixture creation helpers (`user`, `patient`, `clinic`, `appointment`)
- supports authenticated and HTMX-like request execution helpers

## SQLite Test Modes

By default, tests use file-backed SQLite for local stability.

Optional in-memory mode is available for faster CI or local runs:

```bash
MICON_TEST_SQLITE_INMEMORY=1 go test ./...
```

When enabled, harness uses shared in-memory SQLite settings (`mode=memory`,
`cache=shared`) and constrains SQL connections for consistent behavior.

## CI Expectations

The CI gate runs:

- `go test ./...`
- `go test -race ./...`

This keeps route regressions and concurrency issues visible before merge.

## Next Expansion

- Replace low-value generic handler tests with service-focused handler tests.
- Add `handlers_test.go` suites incrementally per service package.
- Keep test names behavior-oriented and persistence-aware.
