# Testing Strategy

This project uses a layered testing approach focused on regression prevention,
route behavior confidence, and safe iterative refactors.

## Goals

- Catch high-risk route regressions early.
- Validate persistence behavior, not only HTTP responses.
- Keep tests readable and maintainable as the app grows.

## Current Baseline

- Reusable integration harness: `tests/helpers_test.go`
- Cross-service regression coverage: `tests/regression_routes_test.go`
- Service-focused handler suites in `tests/*_handlers_test.go`

Covered regressions currently include:

- `/appointments/search/clinics` route behavior
- cross-service patch/delete flows for patient and appointment

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

## Coverage Baseline

Current global baseline (2026-03-09):

- `total: 0.2%` (`go tool cover -func=coverage/c.out`)

Generate and review coverage locally:

```bash
go test ./... -coverprofile=coverage/c.out
go tool cover -func=coverage/c.out
```

Notes:

- This baseline is informational for now; no coverage threshold gate is enforced yet.
- Next step is to introduce phased coverage gates in CI (global first, then package-level targets).

## Testing Maintenance Loop

- Any bugfix touching handlers/routes must add or update at least one regression test in the corresponding `tests/<service>_handlers_test.go` file.
- If a handler/route change ships without test updates, the PR description must include a short rationale.

## Handler Suite Migration Checklist

Use this checklist when replacing broad route tests with service-focused handler
test suites.

- Create `tests/<service>_handlers_test.go` and group route behavior by
  handler intent.
- Move service-specific assertions out of
  `tests/regression_routes_test.go` once equivalent coverage exists.
- Keep cross-service/regression scenarios in
  `tests/regression_routes_test.go`.
- Assert both response behavior (status, redirects, HTMX headers) and
  persistence side effects where relevant.
- Include auth/role-gating cases for protected routes.
- Run verification before merging:
  - `go test ./tests -run Test<Service>Handlers -count=1`
  - `go test ./...`
  - `go test -race ./...`

### Migration Status

- Done: `appointment`, `patient`, `clinic`, `user`, `theme`, `dashboard`,
  `admin`
- Pending: `auth`

### Definition of Done

- No duplicated service-specific assertions remain in
  `tests/regression_routes_test.go`.
- Handler suite naming/style is consistent (`tests/<service>_handlers_test.go`).
- CI gate is green (`go test ./...` and `go test -race ./...`).

## Negative Path Matrix (Appointment + Clinic)

These status expectations are covered in handler suites and should remain
consistent unless route contracts change.

| Endpoint Class | 400 | 401 | 403 | 404 | 422 |
| --- | --- | --- | --- | --- | --- |
| Appointment update/create flows | malformed bind/input | unauthenticated JSON clients | n/a | unknown or cross-user record | invalid create payload (missing required relationships) |
| Clinic update/delete/search flows | malformed bind/short search term | unauthenticated JSON clients | n/a | unknown or cross-user record | invalid clinic boundaries |

Exceptions:

- `403` is not expected on appointment/clinic routes because these routes are
  auth-gated (`MustAuthenticate`) but not role-gated (`MustBeAdmin`).

## Next Expansion

- Replace low-value generic handler tests with service-focused handler tests.
- Add `handlers_test.go` suites incrementally per service package.
- Keep test names behavior-oriented and persistence-aware.
