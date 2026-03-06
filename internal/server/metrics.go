package server

import (
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	registerMetricsOnce sync.Once

	httpRequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "HTTP request duration in seconds.",
			Buckets: []float64{
				0.005, 0.01, 0.025, 0.05, 0.075,
				0.1, 0.15, 0.2, 0.3, 0.5,
				0.75, 1, 1.5, 2, 3, 5, 10,
			},
		},
		[]string{"route", "method", "status_group"},
	)

	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"route", "method", "status_group"},
	)
)

func requestMetricsMiddleware() func(c fiber.Ctx) error {
	registerMetricsOnce.Do(func() {
		prometheus.MustRegister(httpRequestDurationSeconds, httpRequestsTotal)
	})

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
		httpRequestDurationSeconds.WithLabelValues(routePath, method, statusGroup).Observe(elapsedSeconds)
		httpRequestsTotal.WithLabelValues(routePath, method, statusGroup).Inc()

		return err
	}
}

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
