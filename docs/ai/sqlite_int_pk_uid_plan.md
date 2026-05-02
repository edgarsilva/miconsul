# SQLite Integer PK + UID Plan

## Goal
- Move all core models and schema tables to `id INTEGER PRIMARY KEY`.
- Keep prefixed public IDs as `uid TEXT NOT NULL UNIQUE`.
- Use integer foreign keys internally for joins and indexes.
- Keep external/API references stable by using `uid` in routes and request payloads.

## Git/Branch Checklist
- [x] Stash local unrelated change (`internal/services/dashboard/service.go`).
- [x] Fetch and prune remotes.
- [x] Create new branch from `origin/main`: `feat/sqlite-int-pk-uid`.
- [ ] Optionally reconcile local `main` divergence after this feature ships.

## Implementation Checklist

### 1) Model Layer
- [ ] Update model identity fields to `ID uint` + `UID string`.
- [ ] Update `BeforeCreate` hooks to assign `UID` instead of `ID`.
- [ ] Update relation FK fields to integer IDs where appropriate.
- [ ] Keep model-specific UID prefixes (`user`, `clnc`, `ptnt`, etc.).

### 2) Query/Service Layer
- [ ] Switch external lookups from `id` to `uid`.
- [ ] Keep internal joins and relation traversal on integer `id`.
- [ ] Ensure enqueue/job payload semantics remain external-UID-safe.

### 3) Migrations/Schema
- [ ] Rewrite bootstrap schema for integer PK + UID unique constraints.
- [ ] Keep final columns/indexes expected by current app behavior.
- [ ] Ensure FK constraints point to integer PK columns.
- [ ] Keep FTS setup and triggers compatible with new keys.

### 4) Tests/Seeders
- [ ] Update tests expecting string IDs.
- [ ] Update seeders/fixtures to use UID for external selection.
- [ ] Re-run migrations and DB bootstrap tests.

### 5) Validation
- [ ] Run `go test ./...`.
- [ ] Run `make docker/image`.
- [ ] Report files changed and remaining follow-ups.

## Notes
- This branch assumes no production data migration is required.
- If data appears before merge, this plan must switch to additive/online migration.
