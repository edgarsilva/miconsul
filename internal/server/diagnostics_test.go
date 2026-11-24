package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
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
