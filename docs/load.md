# Load and memory tests

Run on demand against an isolated stack. Nothing here runs in CI. The stack has jobs, outbound messaging and the rate limiter off, and its databases are deleted by `make load/down`.

Needs `k6`, `oha` and `jq` on the PATH (`mise` installs the first two).

## Commands

| Command | Does |
|---|---|
| `make load/up` | Builds the app and starts it on port 3200 with the baseline seed and bulk data (`scripts/stack.sh up --bulk`). |
| `make load/oha` | `oha` against `/appointments/` for 30s at concurrency 30. Tune with `TARGET_PATH`, `DURATION`, `CONCURRENCY`. |
| `make load/k6` | The `k6` mix in `scripts/load/scenario.js`: signs in as the seed admin, then 25% dashboard, 25% appointments (month), 15% clinic search, 15% patient search, 10% revenue and 10% appointment writes (create or update). Tune with `VUS` and `DURATION`. Fails when `http_req_failed` is 1% or more, p95 is 500ms or more, or checks drop under 99%. |
| `make load/run` | The `k6` mix while `scripts/load/memory.sh` polls `/api/admin/runtime` every 2s into `tmp/load/memory.csv`, then prints peak and end live heap, the heap floor of the first and last third of the run, peak goroutines and GC cycles. |
| `make load/down` | Stops the stack and deletes its data. |

`LOAD_URL` points the targets at another instance. `scripts/load/memory.sh report <csv>` reads any sampler file.

A rising heap floor across a run means memory is being retained; a flat or falling floor means the live heap returns to its baseline.

## Baselines

Seed data: `--bulk` (5 users, 20 clinics, 60 patients, 120 future and 300 past appointments), 20 VUs, 30s, on a development laptop, 2026-11-25.

| Run | Result |
|---|---|
| `make load/oha` on a fresh stack | 11,062 requests/s, average 2.7ms, no failures |
| `make load/k6` on a fresh stack (10s) | 1,477 iterations/s, p95 31ms |
| `make load/run`, second run on the same stack (2,929 appointments after the writes) | 590 iterations/s, p95 81ms, live heap floor 14.8 MiB to 13.5 MiB, peak 26.9 MiB, 64 goroutines at the end |

Throughput falls and latency rises across runs because the `k6` mix creates appointments and the month view lists all of them. Restart with `make load/up` for comparable numbers.
