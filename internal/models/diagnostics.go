package models

import "time"

// RuntimeStats is a snapshot of the running process: how long it has been up,
// its goroutines and its memory use in bytes.
type RuntimeStats struct {
	Uptime          time.Duration
	Goroutines      int
	AllocBytes      uint64
	TotalAllocBytes uint64
	SysBytes        uint64
	HeapObjects     uint64
}

// HealthReport is the result of the liveness, readiness and startup probes
// and of every registered dependency check, with the server's boot timings.
type HealthReport struct {
	StartedAt         time.Time
	ReadyAt           time.Time
	BootstrapDuration time.Duration
	Uptime            time.Duration
	Version           string
	Environment       string
	Livez             bool
	Readyz            bool
	Startupz          bool
	Dependencies      []HealthDependency
}

// HealthDependency is the result of one dependency check: whether it passed,
// how long it took and the error when it failed.
type HealthDependency struct {
	Name    string
	OK      bool
	Latency time.Duration
	Error   string
}

// OK reports whether every probe and every dependency check passed.
func (r HealthReport) OK() bool {
	if !r.Livez || !r.Readyz || !r.Startupz {
		return false
	}

	for _, dep := range r.Dependencies {
		if !dep.OK {
			return false
		}
	}

	return true
}
