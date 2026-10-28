package server

import (
	"context"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v3"
)

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
	dependencies, dependenciesOK := s.runHealthChecks(c.Context())

	status := "ok"
	httpStatus := fiber.StatusOK
	if !livez || !readyz || !startupz || !dependenciesOK {
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
			Livez:        livez,
			Readyz:       readyz,
			Startupz:     startupz,
			Dependencies: dependencies,
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
	Livez        bool                              `json:"livez"`
	Readyz       bool                              `json:"readyz"`
	Startupz     bool                              `json:"startupz"`
	Dependencies map[string]debugDependencyPayload `json:"dependencies"`
}

type debugDependencyPayload struct {
	OK        bool   `json:"ok"`
	LatencyMS int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

type healthCheck struct {
	name  string
	check func(context.Context) error
}

// AddHealthCheck registers a dependency check that /debug/health runs and
// reports. Register checks at startup, before the server serves requests.
func (s *Server) AddHealthCheck(name string, check func(context.Context) error) {
	s.healthChecks = append(s.healthChecks, healthCheck{name: name, check: check})
}

// runHealthChecks runs the registered dependency checks and reports whether
// all of them passed.
func (s *Server) runHealthChecks(ctx context.Context) (map[string]debugDependencyPayload, bool) {
	results := make(map[string]debugDependencyPayload, len(s.healthChecks))
	allOK := true
	for _, hc := range s.healthChecks {
		start := time.Now()
		err := hc.check(ctx)
		result := debugDependencyPayload{OK: err == nil, LatencyMS: time.Since(start).Milliseconds()}
		if err != nil {
			result.Error = err.Error()
			allOK = false
		}
		results[hc.name] = result
	}

	return results, allOK
}

func timeStringOrEmpty(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.UTC().Format(time.RFC3339)
}
