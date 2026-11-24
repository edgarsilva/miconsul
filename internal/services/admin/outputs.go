package admin

import (
	"time"

	"miconsul/internal/models"
)

type runtimeOutput struct {
	UptimeSeconds   int    `json:"uptime_seconds"`
	Goroutines      int    `json:"goroutines"`
	AllocBytes      uint64 `json:"alloc_bytes"`
	TotalAllocBytes uint64 `json:"total_alloc_bytes"`
	SysBytes        uint64 `json:"sys_bytes"`
	HeapObjects     uint64 `json:"heap_objects"`
	LiveHeapBytes   uint64 `json:"live_heap_bytes"`
	GCCycles        uint32 `json:"gc_cycles"`
}

func newRuntimeOutput(r models.RuntimeStats) runtimeOutput {
	return runtimeOutput{
		UptimeSeconds:   int(r.Uptime.Seconds()),
		Goroutines:      r.Goroutines,
		AllocBytes:      r.AllocBytes,
		TotalAllocBytes: r.TotalAllocBytes,
		SysBytes:        r.SysBytes,
		HeapObjects:     r.HeapObjects,
		LiveHeapBytes:   r.LiveHeapBytes,
		GCCycles:        r.GCCycles,
	}
}

type runtimeHistoryOutput struct {
	IntervalSeconds int                   `json:"interval_seconds"`
	Samples         []runtimeSampleOutput `json:"samples"`
}

type runtimeSampleOutput struct {
	At            string `json:"at"`
	LiveHeapBytes uint64 `json:"live_heap_bytes"`
	Goroutines    int    `json:"goroutines"`
	GCCycles      uint32 `json:"gc_cycles"`
}

func newRuntimeHistoryOutput(interval time.Duration, samples []models.RuntimeSample) runtimeHistoryOutput {
	out := runtimeHistoryOutput{
		IntervalSeconds: int(interval.Seconds()),
		Samples:         make([]runtimeSampleOutput, 0, len(samples)),
	}
	for _, sample := range samples {
		out.Samples = append(out.Samples, runtimeSampleOutput{
			At:            timeStringOrEmpty(sample.At),
			LiveHeapBytes: sample.LiveHeapBytes,
			Goroutines:    sample.Goroutines,
			GCCycles:      sample.GCCycles,
		})
	}

	return out
}

type healthOutput struct {
	Status              string             `json:"status"`
	StartedAt           string             `json:"started_at"`
	ReadyAt             string             `json:"ready_at"`
	BootstrapDurationMS int64              `json:"bootstrap_duration_ms"`
	UptimeSeconds       int                `json:"uptime_seconds"`
	Version             string             `json:"version"`
	Environment         string             `json:"environment"`
	Checks              healthChecksOutput `json:"checks"`
}

type healthChecksOutput struct {
	Livez        bool                        `json:"livez"`
	Readyz       bool                        `json:"readyz"`
	Startupz     bool                        `json:"startupz"`
	Dependencies map[string]dependencyOutput `json:"dependencies"`
}

type dependencyOutput struct {
	OK        bool   `json:"ok"`
	LatencyMS int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

func newHealthOutput(r models.HealthReport) healthOutput {
	status := "ok"
	if !r.OK() {
		status = "degraded"
	}

	dependencies := make(map[string]dependencyOutput, len(r.Dependencies))
	for _, dep := range r.Dependencies {
		dependencies[dep.Name] = dependencyOutput{OK: dep.OK, LatencyMS: dep.Latency.Milliseconds(), Error: dep.Error}
	}

	return healthOutput{
		Status:              status,
		StartedAt:           timeStringOrEmpty(r.StartedAt),
		ReadyAt:             timeStringOrEmpty(r.ReadyAt),
		BootstrapDurationMS: r.BootstrapDuration.Milliseconds(),
		UptimeSeconds:       int(r.Uptime.Seconds()),
		Version:             r.Version,
		Environment:         r.Environment,
		Checks: healthChecksOutput{
			Livez:        r.Livez,
			Readyz:       r.Readyz,
			Startupz:     r.Startupz,
			Dependencies: dependencies,
		},
	}
}

func timeStringOrEmpty(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.UTC().Format(time.RFC3339)
}
