package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"testing/synctest"
	"time"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
)

func TestRuntimeStats(t *testing.T) {
	s := &Server{StartedAt: time.Now().Add(-2 * time.Second)}

	stats := s.RuntimeStats()

	if stats.Uptime < 2*time.Second {
		t.Fatalf("expected uptime of at least 2s, got %v", stats.Uptime)
	}
	runtime.GC()
	stats = s.RuntimeStats()
	if stats.LiveHeapBytes == 0 || stats.GCCycles == 0 {
		t.Fatalf("expected live heap and GC cycles after a collection, got %+v", stats)
	}
	if stats.Goroutines <= 0 || stats.AllocBytes == 0 || stats.SysBytes == 0 {
		t.Fatalf("expected populated runtime stats, got %+v", stats)
	}
}

func TestRuntimeStatsZeroUptimeBeforeStart(t *testing.T) {
	s := &Server{}

	if got := s.RuntimeStats().Uptime; got != 0 {
		t.Fatalf("expected zero uptime without StartedAt, got %v", got)
	}
}

func TestHealthReport(t *testing.T) {
	s := healthyServer(t)

	report := reportFor(t, s)

	if !report.OK() {
		t.Fatalf("expected a healthy report, got %+v", report)
	}
	if report.Version != "1.2.3" || report.Environment != string(appenv.EnvironmentProduction) {
		t.Fatalf("expected version and environment, got %q %q", report.Version, report.Environment)
	}
	if report.BootstrapDuration <= 0 || report.Uptime <= 0 {
		t.Fatalf("expected boot timings, got %+v", report)
	}
	if !report.Livez || !report.Readyz || !report.Startupz {
		t.Fatalf("expected all probes healthy, got %+v", report)
	}
}

func TestHealthReportDegradedWhenReadinessFails(t *testing.T) {
	s := healthyServer(t)
	sqlDB, err := s.DB.SQLDB()
	if err != nil {
		t.Fatalf("get sql db handle: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	report := reportFor(t, s)

	if report.OK() {
		t.Fatalf("expected a degraded report")
	}
	if !report.Livez || report.Readyz || report.Startupz {
		t.Fatalf("expected livez only, got %+v", report)
	}
}

func TestHealthReportDependencies(t *testing.T) {
	tests := []struct {
		name     string
		checkErr error
		wantOK   bool
	}{
		{"passing dependency keeps the report healthy", nil, true},
		{"failing dependency degrades the report", errors.New("logto unreachable"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := healthyServer(t)
			s.AddHealthCheck("logto", func(context.Context) error { return tt.checkErr })

			report := reportFor(t, s)

			if len(report.Dependencies) != 1 || report.Dependencies[0].Name != "logto" {
				t.Fatalf("expected a logto dependency, got %+v", report.Dependencies)
			}
			if got := report.Dependencies[0]; got.OK != tt.wantOK {
				t.Fatalf("expected logto ok=%v, got %+v", tt.wantOK, got)
			}
			if report.OK() != tt.wantOK {
				t.Fatalf("expected report ok=%v, got %v", tt.wantOK, report.OK())
			}
			if !tt.wantOK && report.Dependencies[0].Error == "" {
				t.Fatalf("expected the dependency error in the report")
			}
		})
	}
}

func healthyServer(t *testing.T) *Server {
	t.Helper()

	s := readyServer(t)
	s.App = fiber.New()
	s.Env = &appenv.Env{AppVersion: "1.2.3", Environment: appenv.EnvironmentProduction}
	s.StartedAt = time.Now().Add(-3 * time.Second)
	s.ReadyAt = time.Now().Add(-2 * time.Second)
	s.BootstrapDuration = 500 * time.Millisecond

	return s
}

func reportFor(t *testing.T, s *Server) models.HealthReport {
	t.Helper()

	var report models.HealthReport
	s.App.Get("/report", func(c fiber.Ctx) error {
		report = s.HealthReport(c)
		return nil
	})
	if _, err := s.App.Test(httptest.NewRequest(http.MethodGet, "/report", nil)); err != nil {
		t.Fatalf("health report request failed: %v", err)
	}

	return report
}

func TestRuntimeSamplerRecordsSamplesUntilCancelled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := &Server{}
		ctx, cancel := context.WithCancel(t.Context())

		if err := s.StartRuntimeSampler(ctx, time.Second, 3); err != nil {
			t.Fatalf("start sampler: %v", err)
		}
		synctest.Wait()
		if got := len(s.RuntimeHistory()); got != 1 {
			t.Fatalf("expected an immediate first sample, got %d", got)
		}

		time.Sleep(10*time.Second + time.Second/2)
		synctest.Wait()
		history := s.RuntimeHistory()
		if len(history) != 3 {
			t.Fatalf("expected the history capped at 3 samples, got %d", len(history))
		}
		for i := 1; i < len(history); i++ {
			if !history[i].At.After(history[i-1].At) {
				t.Fatalf("expected samples oldest first, got %v then %v", history[i-1].At, history[i].At)
			}
		}

		cancel()
		synctest.Wait()
		before := len(s.RuntimeHistory())
		time.Sleep(5 * time.Second)
		synctest.Wait()
		if got := len(s.RuntimeHistory()); got != before {
			t.Fatalf("expected the sampler to stop after cancel, history went from %d to %d", before, got)
		}
	})
}

func TestStartRuntimeSamplerRejectsInvalidArguments(t *testing.T) {
	s := &Server{}

	if err := s.StartRuntimeSampler(t.Context(), 0, 10); err == nil {
		t.Fatal("expected an error for a zero interval")
	}
	if err := s.StartRuntimeSampler(t.Context(), time.Second, 0); err == nil {
		t.Fatal("expected an error for a zero size")
	}
}

func TestRuntimeHistoryIsEmptyBeforeTheSamplerStarts(t *testing.T) {
	s := &Server{}

	if got := s.RuntimeHistory(); got == nil || len(got) != 0 {
		t.Fatalf("expected an empty, non-nil history, got %v", got)
	}
}
