package models

import "time"

// RuntimeStats is a snapshot of the running process: how long it has been up,
// its goroutines and its memory use in bytes. HeapObjects counts garbage the
// collector has not freed yet, so it rises between collections; LiveHeapBytes
// is the heap the last collection found in use, and a floor that keeps rising
// across collections is what a leak looks like.
type RuntimeStats struct {
	Uptime          time.Duration
	Goroutines      int
	AllocBytes      uint64
	TotalAllocBytes uint64
	SysBytes        uint64
	HeapObjects     uint64
	LiveHeapBytes   uint64
	GCCycles        uint32
}

// HeapInUsePercent returns the heap in use as a whole percent of the memory
// reserved from the OS, rounded half up, e.g. 25 MiB of 100 MiB -> 25.
func (r RuntimeStats) HeapInUsePercent() int {
	return sharePercent(int(r.AllocBytes), int(r.SysBytes))
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

// FailingChecks returns how many probes and dependency checks failed,
// e.g. readiness and one dependency failing -> 2.
func (r HealthReport) FailingChecks() int {
	failing := 0
	for _, ok := range []bool{r.Livez, r.Readyz, r.Startupz} {
		if !ok {
			failing++
		}
	}

	for _, dep := range r.Dependencies {
		if !dep.OK {
			failing++
		}
	}

	return failing
}
