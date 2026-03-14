# PR Checklist - Asynq Migration (Appointments) + Queue UI

## Phase 0 - Branch + Baseline
- [ ] Confirm clean tree (`git status --short --branch`)
- [ ] Sync main (`git fetch origin && git checkout main && git pull --ff-only origin main`)
- [ ] Create feature branch (`git checkout -b feat/asynq-appointments-queue-ui`)
- [ ] Capture baseline behavior (booked alert + reminder flow)
- [ ] Baseline tests green (`go test ./...`)

### Exit Criteria
- [ ] Branch is created from up-to-date `main`
- [ ] Baseline behavior is documented and reproducible

### Commit
- [ ] No commit required (setup only)

---

## Phase 1 - Git Housekeeping (First)
- [ ] Confirm branch tracking and divergence (`git status --short --branch`)
- [ ] Review working tree for accidental/unrelated files (`git status --short`)
- [ ] Keep only intended feature files for this workstream
- [ ] Verify no unintended generated/binary artifacts are present
- [ ] Record housekeeping note in PR description

### Exit Criteria
- [ ] Working tree scope is clean and intentional
- [ ] Housekeeping note is added to PR description

### Commit
- [ ] `chore(git): housekeeping for asynq migration branch` (only if repo files actually changed)

---

## Phase 2 - Queue Infra Wiring (No Appointment Behavior Change)
- [ ] Add dependency `github.com/hibiken/asynq`
- [ ] Add queue config to `internal/lib/appenv` (host, port, password, db, UI toggle/path)
- [ ] Add Valkey service in `docker-compose.yaml`
- [ ] Wire Asynq client/server/scheduler startup and graceful shutdown in `cmd/app/main.go`
- [ ] Keep old cron/worker active during transition

### Exit Criteria
- [ ] App boots with queue infra enabled
- [ ] No appointment flow behavior changes yet

### Commit
- [ ] `chore(queue): add asynq/valkey config and bootstrap wiring`

---

## Phase 3 - Queue Module Foundation
- [ ] Create `internal/lib/queue` package
- [ ] Define task types/constants
- [ ] Define payload structs
- [ ] Add enqueue helpers (`EnqueueBookedAlert`, `EnqueueReminderAlert`, etc.)
- [ ] Add server mux/handler registration
- [ ] Add scheduler registration entrypoint

### Exit Criteria
- [ ] Queue module compiles and supports enqueue/consume path

### Commit
- [ ] `feat(queue): add queue task contracts enqueue and handler registration`

---

## Phase 4 - Booked Alert Migration
- [ ] Replace `SendToWorker(...)` booked-alert path with Asynq enqueue in `internal/services/appointment/alerts.go`
- [ ] Add booked-alert task handler
- [ ] Preserve DB semantics (`BookedAlertSentAt` + alert status row)
- [ ] Add idempotency guard (no-op if already sent)

### Exit Criteria
- [ ] Appointment creation enqueues durable booked-alert task
- [ ] Duplicate executions do not double-send

### Commit
- [ ] `refactor(appointment): move booked alert dispatch to asynq`

---

## Phase 5 - Reminder Pipeline Migration
- [ ] Replace `RegisterCronJob` reminder scan with Asynq scheduler task (every 1m)
- [ ] Sweep task enqueues per-appointment reminder tasks
- [ ] Preserve DB semantics (`ReminderAlertSentAt` + alert status rows)
- [ ] Add idempotency guard in reminder handler

### Exit Criteria
- [ ] Reminder discovery and dispatch run via Asynq scheduler/tasks
- [ ] Retries/re-runs do not create duplicate sends

### Commit
- [ ] `refactor(appointment): migrate reminder scan and sends to asynq scheduler`

---

## Phase 6 - Queue Web UI (Embedded Admin Route)
- [ ] Expose Asynq monitor UI at `/debug/queue`
- [ ] Protect route with `MustBeAdmin`
- [ ] Gate by env/config toggle for non-local environments

### Exit Criteria
- [ ] Admin can inspect active/scheduled/retry/dead tasks in app route

### Commit
- [ ] `feat(debug): add admin-protected asynq ui route`

---

## Phase 7 - Remove Legacy Paths
- [ ] Remove Ants worker wiring that is no longer used
- [ ] Remove legacy appointment cron registration
- [ ] Clean dead code and stale bootstrap hooks

### Exit Criteria
- [ ] Single async path remains (Asynq + Valkey)

### Commit
- [ ] `chore(appointment): remove legacy cron and worker paths`

---

## Phase 8 - Tests, Docs, Verification
- [ ] Update tests that assumed cron/Ants behavior
- [ ] Add queue-focused tests (enqueue path + idempotent handlers)
- [ ] Update docs/runbook (Valkey startup, queue UI, retry/dead task inspection)
- [ ] Regenerate templ only if `.templ` files changed (`make templ`)
- [ ] Run full tests (`go test ./...`)

### Exit Criteria
- [ ] Acceptance criteria are met
- [ ] CI/local test suite is green

### Commit
- [ ] `test/docs(queue): update async tests and queue operations runbook`

---

## Final Acceptance Checklist
- [ ] `/debug/queue` monitor route works and is admin-protected
- [ ] New appointment enqueue creates durable booked-alert task
- [ ] Reminder sweep runs via Asynq scheduler and enqueues reminder tasks
- [ ] Job processing updates alert status + sent timestamps correctly
- [ ] Queue survives app restarts with Valkey backend
- [ ] `go test ./...` passes
