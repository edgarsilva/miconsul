# Observability Runbook

This runbook captures the baseline commands and queries used to validate the
local LGTM setup (Loki, Tempo, Prometheus, Grafana).

## Prerequisites

- App running locally on `http://localhost:3000`
- LGTM stack running (`make docker/up`)
- Grafana available on `http://localhost:3001`

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

## Metrics Queries (Prometheus)

All examples assume `job="miconsul"`.

- Total RPS:

```promql
sum(rate(http_requests_total{job="miconsul"}[5m]))
```

- Total RPM:

```promql
60 * sum(rate(http_requests_total{job="miconsul"}[5m]))
```

- RPM by status group:

```promql
60 * sum by (status_group) (rate(http_requests_total{job="miconsul"}[5m]))
```

- Per-route RPS:

```promql
sum by (route) (rate(http_requests_total{job="miconsul"}[5m]))
```

- Per-route requests over last 5m (count/window):

```promql
sum by (route) (increase(http_requests_total{job="miconsul"}[5m]))
```

- Overall p95 latency (ms):

```promql
1000 * histogram_quantile(
  0.95,
  sum by (le) (rate(http_request_duration_seconds_bucket{job="miconsul"}[5m]))
)
```

- Per-route p95 latency (ms):

```promql
1000 * histogram_quantile(
  0.95,
  sum by (le, route) (rate(http_request_duration_seconds_bucket{job="miconsul"}[5m]))
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
- `http_request_duration_seconds*` -> seconds
- `1000 * ...duration_seconds...` -> milliseconds
