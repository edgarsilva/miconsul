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

// requestMetricsMiddleware records HTTP request metrics using both pull and push paths.
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
