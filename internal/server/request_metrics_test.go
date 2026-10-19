package server

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"miconsul/internal/lib/appenv"

	"github.com/gofiber/fiber/v3"
)

func TestHandleDebugRuntime(t *testing.T) {
	s := &Server{StartedAt: time.Now().Add(-2 * time.Second)}
	app := fiber.New()
	app.Get("/debug/runtime", s.HandleDebugRuntime)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/debug/runtime", nil))
	if err != nil {
		t.Fatalf("runtime request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

type debugHealthDetailsResponse struct {
	Status              string `json:"status"`
	StartedAt           string `json:"started_at"`
	ReadyAt             string `json:"ready_at"`
	BootstrapDurationMS int64  `json:"bootstrap_duration_ms"`
	UptimeSeconds       int    `json:"uptime_seconds"`
	Version             string `json:"version"`
	Environment         string `json:"environment"`
	Checks              struct {
		Livez    bool `json:"livez"`
		Readyz   bool `json:"readyz"`
		Startupz bool `json:"startupz"`
	} `json:"checks"`
}

func TestHandleDebugHealthDetails(t *testing.T) {
	s := readyServer(t)
	s.App = fiber.New()
	s.Env = &appenv.Env{AppVersion: "1.2.3", Environment: appenv.EnvironmentProduction}
	s.StartedAt = time.Now().Add(-3 * time.Second)
	s.ReadyAt = time.Now().Add(-2 * time.Second)
	s.BootstrapDuration = 500 * time.Millisecond

	s.App.Get("/debug/health", s.HandleDebugHealthDetails)

	resp, err := s.App.Test(httptest.NewRequest(http.MethodGet, "/debug/health", nil))
	if err != nil {
		t.Fatalf("health details request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var payload debugHealthDetailsResponse
	if err := json.UnmarshalRead(resp.Body, &payload); err != nil {
		t.Fatalf("decode response payload: %v", err)
	}

	if payload.Status != "ok" {
		t.Fatalf("expected status ok, got %q", payload.Status)
	}
	if payload.StartedAt == "" || payload.ReadyAt == "" {
		t.Fatalf("expected started_at and ready_at to be populated")
	}
	if payload.BootstrapDurationMS <= 0 {
		t.Fatalf("expected bootstrap_duration_ms to be > 0")
	}
	if payload.UptimeSeconds <= 0 {
		t.Fatalf("expected uptime_seconds to be > 0")
	}
	if payload.Version != "1.2.3" {
		t.Fatalf("expected version 1.2.3, got %q", payload.Version)
	}
	if payload.Environment != string(appenv.EnvironmentProduction) {
		t.Fatalf("expected production environment, got %q", payload.Environment)
	}
	if !payload.Checks.Livez || !payload.Checks.Readyz || !payload.Checks.Startupz {
		t.Fatalf("expected all checks healthy, got %+v", payload.Checks)
	}
}

func TestHandleDebugHealthDetailsDegradedWhenReadinessFails(t *testing.T) {
	s := readyServer(t)
	s.App = fiber.New()
	s.Env = &appenv.Env{AppVersion: "1.2.3", Environment: appenv.EnvironmentProduction}
	s.StartedAt = time.Now().Add(-3 * time.Second)
	s.ReadyAt = time.Now().Add(-2 * time.Second)
	s.BootstrapDuration = 500 * time.Millisecond

	sqlDB, err := s.DB.SQLDB()
	if err != nil {
		t.Fatalf("get sql db handle: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	s.App.Get("/debug/health", s.HandleDebugHealthDetails)

	resp, err := s.App.Test(httptest.NewRequest(http.MethodGet, "/debug/health", nil))
	if err != nil {
		t.Fatalf("health details request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.StatusCode)
	}

	var payload debugHealthDetailsResponse
	if err := json.UnmarshalRead(resp.Body, &payload); err != nil {
		t.Fatalf("decode response payload: %v", err)
	}
	if payload.Status != "degraded" {
		t.Fatalf("expected degraded status, got %q", payload.Status)
	}
	if !payload.Checks.Livez {
		t.Fatalf("expected liveness true")
	}
	if payload.Checks.Readyz {
		t.Fatalf("expected readiness false")
	}
	if payload.Checks.Startupz {
		t.Fatalf("expected startup false")
	}
}
