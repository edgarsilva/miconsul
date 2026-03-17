# Jobs Runbook

This runbook covers local operations for the Valkey-backed jobs runtime and the
admin monitor UI.

## Prerequisites

- App running locally on `http://localhost:3000`
- Docker available for local Valkey
- Admin account to access `/admin/jobs`

## Configuration

Set these environment variables (see `.env.example`):

- `JOBS_ENABLED=true`
- `JOBS_UI_ENABLED=true` (for local admin monitor route)
- `VALKEY_HOST=127.0.0.1`
- `VALKEY_PORT=6379`
- `VALKEY_PASSWORD=`
- `VALKEY_DB=0`

## Start Local Infra

Start Valkey (and observability infra used by local workflows):

```bash
make docker/up
```

Optional logs while validating jobs:

```bash
make docker/valkey-logs
```

## Verify Jobs Runtime Boot

1. Start the app (`make dev` or `make run`) with jobs envs enabled.
2. Open `http://localhost:3000/admin/jobs` as an admin user.
3. Confirm monitor sections load (`Queues`, `Scheduled`, `Retry`, `Archived`).

If `/admin/jobs` returns not found, verify `JOBS_UI_ENABLED=true`.

## Operational Checks

### Reminder Sweep

- In `/admin/jobs`, open `Scheduled` and confirm the reminder sweep task exists
  (`appointment:reminder_sweep`) on `@every 1m`.

### Booked Alert Dispatch

- Create a new appointment through the app flow.
- Confirm a durable task is enqueued for `appointment:booked_alert`.

## Retry and Dead Task Inspection

Use `/admin/jobs` to inspect and manage failed tasks:

- `Retry`: tasks waiting for next retry attempt.
- `Archived`: tasks moved after retries are exhausted.

Basic triage flow:

1. Open the failed task payload and inspect `appointment_id`.
2. Confirm related appointment still exists and has required associations.
3. Re-run task from monitor after fixing root cause.
4. If task repeatedly archives, capture payload + error and open a fix issue.

## Restart Survivability Check

To validate persistence across restarts:

1. Ensure `JOBS_ENABLED=true` and Valkey is running.
2. Enqueue booked/reminder tasks via normal app actions.
3. Restart app process only (keep Valkey running).
4. Re-open `/admin/jobs` and confirm queued/scheduled tasks are still visible.
5. Confirm handlers resume processing and timestamps/alerts update.
