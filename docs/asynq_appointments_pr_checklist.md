# PR Checklist - Asynq Migration (Appointments) + Jobs UI

## Phase 0 - Branch + Baseline
- [x] Confirm clean tree (`git status --short --branch`)
- [x] Sync main (`git fetch origin && git checkout main && git pull --ff-only origin main`)
- [x] Create feature branch (`git checkout -b feat/asynq-appointments-queue-ui`)
- [x] Capture baseline behavior (booked alert + reminder flow)
- [x] Baseline tests green (`go test ./...`)

### Exit Criteria
- [x] Branch is created from up-to-date `main`
- [x] Baseline behavior is documented and reproducible

### Commit
- [x] No commit required (setup only)

---

## Phase 1 - Git Housekeeping (First)
- [x] Confirm branch tracking and divergence (`git status --short --branch`)
- [x] Review working tree for accidental/unrelated files (`git status --short`)
- [x] Keep only intended feature files for this workstream
- [x] Verify no unintended generated/binary artifacts are present
- [x] Record housekeeping note in commit trail (PR note optional)

### Exit Criteria
- [x] Working tree scope is clean and intentional
- [x] Housekeeping note is documented in commits/checklist

### Commit
- [x] Housekeeping captured in existing phase commits (no dedicated chore commit needed)

---

## Phase 2 - Jobs Infra Wiring (No Appointment Behavior Change)
- [x] Add dependency `github.com/hibiken/asynq`
- [x] Add jobs/valkey config to `internal/lib/appenv` (`JOBS_*`, `VALKEY_*`)
- [x] Add Valkey service in `docker-compose.yaml`
- [x] Wire Asynq client/server/scheduler startup and graceful shutdown in `cmd/app/main.go`
- [x] Keep old cron/worker active during transition

### Exit Criteria
- [x] App boots with jobs infra enabled
- [x] No appointment flow behavior changes yet

### Commit
- [x] `chore(jobs): add valkey-backed jobs runtime and bootstrap wiring`

---

## Phase 3 - Jobs Module Foundation (Infra-Only)
- [x] Promote jobs package to first-class infra: `internal/jobs`
- [x] Keep jobs package domain-agnostic (no appointment task constants/payloads in infra)
- [x] Add generic enqueue API in jobs runtime (`EnqueueTask`)
- [x] Add server bridge (`s.EnqueueTask(...)`) returning `(jobs.EnqueueInfo, error)`
- [x] Move appointment task contracts to domain package (`internal/services/appointment/tasks.go`)
- [x] Add server mux/handler registration entrypoint in jobs runtime
- [x] Add scheduler registration entrypoint in jobs runtime

### Exit Criteria
- [x] Jobs module compiles and supports enqueue path
- [x] Jobs module supports handler/scheduler registration path

### Commit
- [x] `refactor(jobs): promote infra package and unify enqueue info API`

---

## Phase 4 - Booked Alert Migration
- [ ] Replace `SendToWorker(...)` booked-alert path with `s.EnqueueTask(...)` in `internal/services/appointment/alerts.go`
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
- [x] Replace `RegisterCronJob` reminder scan with Asynq scheduler task (every 1m)
- [x] Sweep task enqueues per-appointment reminder tasks
- [x] Preserve DB semantics (`ReminderAlertSentAt` + alert status rows)
- [x] Add idempotency guard in reminder handler
- [x] Remove appointment cron registration from router wiring once scheduler path is active

### Exit Criteria
- [x] Reminder discovery and dispatch run via Asynq scheduler/tasks
- [x] Retries/re-runs do not create duplicate sends

### Commit
- [x] `refactor(appointment): migrate reminder flow to jobs runtime`

---

## Phase 6 - Jobs Web UI (Embedded Admin Route)
- [x] Expose Asynq monitor UI at `/admin/jobs`
- [x] Protect route with `MustBeAdmin`
- [x] Gate by env/config toggle for non-local environments

### Exit Criteria
- [x] Admin can inspect active/scheduled/retry/dead tasks in app route

### Commit
- [x] `feat(jobs): mount admin monitor UI and schedule reminder sweep`
- [x] `refactor(jobs): centralize monitor handler wiring`

---

## Phase 7 - Remove Legacy Paths
- [x] Keep Ants worker wiring intentionally (retained for non-jobs/background use)
- [x] Remove legacy appointment cron registration
- [x] Clean dead code and stale bootstrap hooks

### Exit Criteria
- [x] Single appointment reminder async path remains (Asynq + Valkey)

### Commit
- [ ] `chore(appointment): remove legacy cron paths and stale wiring`

---

## Phase 8 - Tests, Docs, Verification
- [x] Update tests that assumed cron/Ants behavior
- [x] Add jobs-focused tests (enqueue path + idempotent handlers)
- [ ] Update docs/runbook (Valkey startup, jobs UI, retry/dead task inspection)
- [ ] Regenerate templ only if `.templ` files changed (`make templ`)
- [x] Run full tests (`go test ./...`)

### Exit Criteria
- [ ] Acceptance criteria are met
- [ ] CI/local test suite is green

### Commit
- [ ] `test/docs(jobs): update async tests and jobs operations runbook`

---

## Final Acceptance Checklist
- [x] `/admin/jobs` monitor route works and is admin-protected
- [ ] New appointment enqueue creates durable booked-alert task
- [x] Reminder sweep runs via Asynq scheduler and enqueues reminder tasks
- [x] Job processing updates alert status + sent timestamps correctly
- [ ] Jobs runtime survives app restarts with Valkey backend
- [x] `go test ./...` passes
