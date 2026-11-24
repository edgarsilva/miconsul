package server

import (
	"context"
	"runtime"
	"runtime/metrics"
	"time"

	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
)

type healthCheck struct {
	name  string
	check func(context.Context) error
}

// AddHealthCheck registers a dependency check that HealthReport runs and
// reports. Register checks at startup, before the server serves requests.
func (s *Server) AddHealthCheck(name string, check func(context.Context) error) {
	s.healthChecks = append(s.healthChecks, healthCheck{name: name, check: check})
}

// RuntimeStats returns the process uptime, goroutine count and memory stats.
func (s *Server) RuntimeStats() models.RuntimeStats {
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)

	var uptime time.Duration
	if s != nil && !s.StartedAt.IsZero() {
		uptime = time.Since(s.StartedAt)
	}

	return models.RuntimeStats{
		LiveHeapBytes:   liveHeapBytes(),
		GCCycles:        mem.NumGC,
		Uptime:          uptime,
		Goroutines:      runtime.NumGoroutine(),
		AllocBytes:      mem.Alloc,
		TotalAllocBytes: mem.TotalAlloc,
		SysBytes:        mem.Sys,
		HeapObjects:     mem.HeapObjects,
	}
}

func liveHeapBytes() uint64 {
	sample := []metrics.Sample{{Name: "/gc/heap/live:bytes"}}
	metrics.Read(sample)
	if sample[0].Value.Kind() != metrics.KindUint64 {
		return 0
	}

	return sample[0].Value.Uint64()
}

// HealthReport runs the liveness, readiness and startup probes and every
// registered dependency check.
func (s *Server) HealthReport(c fiber.Ctx) models.HealthReport {
	return models.HealthReport{
		StartedAt:         s.StartedAt,
		ReadyAt:           s.ReadyAt,
		BootstrapDuration: s.BootstrapDuration,
		Uptime:            time.Since(s.StartedAt),
		Version:           s.Env.AppVersion,
		Environment:       string(s.Env.Environment),
		Livez:             livenessProbe(s)(c),
		Readyz:            readinessProbe(s)(c),
		Startupz:          startupProbe(s)(c),
		Dependencies:      s.runHealthChecks(c.Context()),
	}
}

func (s *Server) runHealthChecks(ctx context.Context) []models.HealthDependency {
	results := make([]models.HealthDependency, 0, len(s.healthChecks))
	for _, hc := range s.healthChecks {
		start := time.Now()
		err := hc.check(ctx)
		result := models.HealthDependency{Name: hc.name, OK: err == nil, Latency: time.Since(start)}
		if err != nil {
			result.Error = err.Error()
		}
		results = append(results, result)
	}

	return results
}
