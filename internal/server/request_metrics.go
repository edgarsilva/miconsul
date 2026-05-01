package server

import (
	"runtime"
	"strconv"
	"strings"
	"time"

	obsmetrics "miconsul/internal/observability/metrics"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/attribute"
	metricapi "go.opentelemetry.io/otel/metric"
)

// RequestMetricsMiddleware records HTTP request metrics using both pull and push paths.
func RequestMetricsMiddleware(metrics obsmetrics.HTTPMetrics) func(c fiber.Ctx) error {
	otelHTTPDuration := metrics.HTTPDuration
	otelHTTPRequests := metrics.HTTPRequests
	promHTTPDuration := metrics.PromHTTPDuration
	promHTTPRequests := metrics.PromHTTPRequests

	return func(c fiber.Ctx) error {
		path := c.Path()
		if strings.HasPrefix(path, "/public/") || path == "/favicon.ico" || path == "/metrics" {
			return c.Next()
		}

		startedAt := time.Now()
		err := c.Next()

		routePath := path
		if route := c.Route(); route != nil && route.Path != "" {
			routePath = route.Path
		}

		statusCode := c.Response().StatusCode()
		statusGroup := strconv.Itoa(statusCode/100) + "xx"
		method := c.Method()

		elapsedSeconds := time.Since(startedAt).Seconds()

		// Pull path: exposed at /metrics for Prometheus scraper ingestion.
		if promHTTPDuration != nil {
			promHTTPDuration.WithLabelValues(routePath, method, statusGroup).Observe(elapsedSeconds)
		}
		if promHTTPRequests != nil {
			promHTTPRequests.WithLabelValues(routePath, method, statusGroup).Inc()
		}

		// Push path: emitted via OTLP to the collector/exporter pipeline.
		// Keep labels aligned across pull/push (route, method, status_group) for query parity.
		attrs := []attribute.KeyValue{
			attribute.String("route", routePath),
			attribute.String("method", method),
			attribute.String("status_group", statusGroup),
		}

		if otelHTTPDuration != nil {
			otelHTTPDuration.Record(c.Context(), elapsedSeconds, metricapi.WithAttributes(attrs...))
		}
		if otelHTTPRequests != nil {
			otelHTTPRequests.Add(c.Context(), 1, metricapi.WithAttributes(attrs...))
		}

		return err
	}
}

// HandleDebugRuntime returns runtime process stats for admin-only diagnostics.
func (s *Server) HandleDebugRuntime(c fiber.Ctx) error {
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)

	uptimeSeconds := 0
	if s != nil && !s.StartedAt.IsZero() {
		uptimeSeconds = int(time.Since(s.StartedAt).Seconds())
	}

	return c.JSON(fiber.Map{
		"uptime_seconds":    uptimeSeconds,
		"goroutines":        runtime.NumGoroutine(),
		"alloc_bytes":       mem.Alloc,
		"total_alloc_bytes": mem.TotalAlloc,
		"sys_bytes":         mem.Sys,
		"heap_objects":      mem.HeapObjects,
	})
}

// HandleDebugHealthDetails returns internal health diagnostics for admin-only usage.
func (s *Server) HandleDebugHealthDetails(c fiber.Ctx) error {
	livez := livenessProbe(s)(c)
	readyz := readinessProbe(s)(c)
	startupz := startupProbe(s)(c)

	status := "ok"
	httpStatus := fiber.StatusOK
	if !livez || !readyz || !startupz {
		status = "degraded"
		httpStatus = fiber.StatusServiceUnavailable
	}

	payload := debugHealthDetailsPayload{
		Status:              status,
		StartedAt:           timeStringOrEmpty(s.StartedAt),
		ReadyAt:             timeStringOrEmpty(s.ReadyAt),
		BootstrapDurationMS: s.BootstrapDuration.Milliseconds(),
		UptimeSeconds:       int(time.Since(s.StartedAt).Seconds()),
		Version:             s.Env.AppVersion,
		Environment:         string(s.Env.Environment),
		Checks: debugHealthChecksPayload{
			Livez:    livez,
			Readyz:   readyz,
			Startupz: startupz,
		},
	}

	return c.Status(httpStatus).JSON(payload)
}

type debugHealthDetailsPayload struct {
	Status              string                   `json:"status"`
	StartedAt           string                   `json:"started_at"`
	ReadyAt             string                   `json:"ready_at"`
	BootstrapDurationMS int64                    `json:"bootstrap_duration_ms"`
	UptimeSeconds       int                      `json:"uptime_seconds"`
	Version             string                   `json:"version"`
	Environment         string                   `json:"environment"`
	Checks              debugHealthChecksPayload `json:"checks"`
}

type debugHealthChecksPayload struct {
	Livez    bool `json:"livez"`
	Readyz   bool `json:"readyz"`
	Startupz bool `json:"startupz"`
}

func timeStringOrEmpty(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.UTC().Format(time.RFC3339)
}
