# Observability Runbook

This runbook captures the baseline commands and queries used to validate the
local LGTM setup (Loki, Grafana, Tempo, Prometheus(Mimir too if needed)).

## Prerequisites

- App running locally on `http://localhost:3000`
- LGTM stack running (`make docker/up`)
- Grafana available on `http://localhost:3080` (override with `GRAFANA_PORT`)

## Health Probe Monitoring Profile (Uptime Kuma)

Use HTTP(s) monitors with follow-redirects disabled and response status checks
enabled.

- `/livez`
  - interval: `30s`
  - timeout: `5s`
  - retries: `2`
  - intent: process uptime and HTTP serving path
- `/readyz`
  - interval: `45s`
  - timeout: `5s`
  - retries: `3`
  - intent: traffic readiness (includes DB query path)
- `/startupz` (optional)
  - interval: `2m`
  - timeout: `5s`
  - retries: `1`
  - intent: post-restart bootstrap confirmation

Alert routing labels:

- `probe=livez` for uptime incidents
- `probe=readyz` for dependency/service health incidents
- `probe=startupz` for restart/bootstrap incidents

Suggested trigger conditions:

- `readyz`: alert on 3 consecutive failures
- `livez`: alert on any failure sustained for 1-2 checks

## Local Probe Verification

Run focused probe tests:

```bash
go test ./internal/server -run 'Test(LivenessReadinessStartupProbes|ReadinessProbeFailsWhenDatabaseClosed|ReadinessProbeFailsWhenDatabaseIsLockedBeyondProbeTimeout)'
```

What these scenarios validate:

- startup/restart path: startup probe transitions to healthy after grace period
- DB down path: readiness fails when the DB handle is closed
- slow/blocked DB path: readiness fails while DB is lock-blocked beyond probe timeout

## Debug Health Details Contract

Internal diagnostics endpoint:

- `GET /debug/health` (admin-only)

Contract guarantees:

- always returns the same JSON keys (stable schema)
- avoids `null` values
- timestamps are RFC3339 UTC strings; empty string means unavailable

Response keys:

- `status`
- `started_at`
- `ready_at`
- `bootstrap_duration_ms`
- `uptime_seconds`
- `version`
- `environment`
- `checks.livez`
- `checks.readyz`
- `checks.startupz`

## Troubleshooting: `readyz` Fails While `livez` Passes

Interpretation:

- the process is up (`/livez`), but dependency-readiness path is degraded (`/readyz`)
- in this app, `/readyz` includes a DB round-trip (`SELECT 1`), so DB availability/latency/lock pressure is the first suspect

Quick triage:

```bash
curl -sS -o /dev/null -w "livez:%{http_code}\n" http://localhost:3000/livez
curl -sS -o /dev/null -w "readyz:%{http_code}\n" http://localhost:3000/readyz
curl -sS http://localhost:3000/debug/health -H "Authorization: Bearer <ADMIN_JWT>" | jq
```

What to check next:

- logs: filter `event=http_request` with `route=/readyz` and inspect `status`, `duration_ms`, and `error`
- DB health: verify DB file/path permissions, lock contention, and recent migration activity
- startup baseline: inspect `event=server_startup` and compare `bootstrap_duration_ms` against normal deploys

Recovery actions:

- if DB is unavailable or locked, restore DB availability first; app restart alone may not fix root cause
- if DB recovered but readiness still fails, restart app and re-check `/readyz` and `/debug/health`
- escalate when `/readyz` remains failing for 3+ checks or latency remains elevated after dependency recovery

## Telemetry Configuration

The app emits traces, metrics, and logs through OpenTelemetry when OTLP is
configured.

Core environment variables:

- `OTEL_EXPORTER_OTLP_ENDPOINT` (required to enable OTLP export)
- `OTEL_EXPORTER_OTLP_INSECURE` (optional; useful for local docker stack)
- `OTEL_SERVICE_NAME` (defaults to `miconsul`)
- `OTEL_TRACER_SERVER` (defaults to `miconsul.server`)
- `OTEL_TRACER_AUTH` (defaults to `miconsul.auth`)

Without `OTEL_EXPORTER_OTLP_ENDPOINT`, the trace, metric, and log providers are
no-ops outside production: app behavior is unchanged but nothing is exported.
In production (`APP_ENV=production`) the endpoint is required and startup fails
without it. If the collector is unreachable at runtime, the app keeps serving and
OTel logs export errors to stderr.

## Signals Emitted by App

- HTTP metrics: the OpenTelemetry semconv `http.server.request.duration`
  histogram (plus request/response body size and active requests), recorded
  by the Fiber OTEL middleware and pushed over OTLP. Prometheus stores it as
  `http_server_request_duration_seconds_*`. There is no `/metrics` scrape
  endpoint.
- HTTP logs (`event=http_request`): emitted by request logging middleware.
- DB logs (`event=db_query`): emitted by GORM logger wrapper.
- Traces: emitted by Fiber OTEL middleware and service spans.

Useful log fields for correlation:

- `trace_id`
- `route`
- `status`
- `duration_ms`
- `db_operation`

## Traffic Generation

Use steady synthetic traffic for dashboards:

```bash
make obs/load/light
make obs/load/medium
make obs/load/heavy
```

Run an authenticated benchmark load test:

```bash
make load/test
```

## Quick Query Pack

After 1-2 minutes of smoke traffic, these three queries should return data.

- Metrics (Prometheus):

```promql
sum(rate(http_server_request_duration_seconds_count{job="miconsul"}[5m]))
```

Expected result: non-empty series, typically `> 0` after `make obs/load/light`
for ~2 minutes.

- Logs (Loki):

```logql
{service_name="miconsul"} | event=`http_request`
```

Expected result: recent `http_request` log lines within the last 2-5 minutes
after smoke traffic starts.

- Traces (Tempo):

```traceql
{ resource.service.name = "miconsul" }
```

Expected result: at least one trace for `miconsul` in the selected time range
after smoke traffic starts.

## Metrics Queries (Prometheus)

All examples assume `job="miconsul"`.

Request counts come from the histogram's `_count` series. Labels follow the
OpenTelemetry HTTP semantic conventions: `http_request_method`, `http_route`
(absent when no route matched), and `http_response_status_code`.

- Total RPS:

```promql
sum(rate(http_server_request_duration_seconds_count{job="miconsul"}[5m]))
```

- Total RPM:

```promql
60 * sum(rate(http_server_request_duration_seconds_count{job="miconsul"}[5m]))
```

- RPM by status code:

```promql
60 * sum by (http_response_status_code) (rate(http_server_request_duration_seconds_count{job="miconsul"}[5m]))
```

- 5xx RPM:

```promql
60 * sum(rate(http_server_request_duration_seconds_count{job="miconsul", http_response_status_code=~"5.."}[5m]))
```

- Per-route RPS:

```promql
sum by (http_route, http_request_method) (rate(http_server_request_duration_seconds_count{job="miconsul"}[5m]))
```

- Per-route requests over last 5m (count/window):

```promql
sum by (http_route) (increase(http_server_request_duration_seconds_count{job="miconsul"}[5m]))
```

- Overall p95 latency:

```promql
histogram_quantile(
  0.95,
  sum by (le) (rate(http_server_request_duration_seconds_bucket{job="miconsul"}[5m]))
)
```

- Per-route p95 latency:

```promql
histogram_quantile(
  0.95,
  sum by (le, http_route) (rate(http_server_request_duration_seconds_bucket{job="miconsul"}[5m]))
)
```

## Logs Queries (Loki)

- All app logs:

```logql
{service_name="miconsul"}
```

- HTTP request events:

```logql
{service_name="miconsul"} | event=`http_request`
```

- DB query events:

```logql
{service_name="miconsul"} | event=`db_query`
```

- HTTP route filter:

```logql
{service_name="miconsul"} | event=`http_request` | route=`/appointments/`
```

- DB operation filter:

```logql
{service_name="miconsul"} | event=`db_query` | db_operation=`SELECT`
```

- Correlate logs by trace id:

```logql
{service_name="miconsul"} | trace_id=`<trace_id>`
```

## Traces (Tempo)

- Service traces:

```traceql
{ resource.service.name = "miconsul" }
```

Use `trace_id` from logs to jump into the trace in Grafana Explore.

## Units Reference

- `rate(counter[5m])` -> requests/second (`req/s`)
- `60 * rate(...)` -> requests/minute (`req/min`)
- `increase(counter[5m])` -> total count during the 5-minute window
- `http_server_request_duration_seconds*` -> seconds
- `1000 * ...duration_seconds...` -> milliseconds ([!NOTE] No longer needed grafana shows them correctly)
