package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"miconsul/internal/lib/appenv"
	obslogging "miconsul/internal/observability/logging"
	obsmetrics "miconsul/internal/observability/metrics"

	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus"
)

func TestRequestMetricsMiddlewareBranches(t *testing.T) {
	metrics := obsmetrics.HTTPMetrics{
		PromHTTPDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{Name: "test_http_duration_seconds", Help: "test duration"},
			[]string{"route", "method", "status_group"},
		),
		PromHTTPRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "test_http_requests_total", Help: "test requests"},
			[]string{"route", "method", "status_group"},
		),
	}

	app := fiber.New()
	app.Use(RequestMetricsMiddleware(metrics))
	app.Get("/ok", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusCreated)
	})
	app.Get("/public/ping", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/ok", nil))
	if err != nil || resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("/ok request failed status=%d err=%v", resp.StatusCode, err)
	}

	resp, err = app.Test(httptest.NewRequest(http.MethodGet, "/public/ping", nil))
	if err != nil || resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("/public request failed status=%d err=%v", resp.StatusCode, err)
	}
}

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

	s.App.Get("/debug/health/details", s.HandleDebugHealthDetails)

	resp, err := s.App.Test(httptest.NewRequest(http.MethodGet, "/debug/health/details", nil))
	if err != nil {
		t.Fatalf("health details request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var payload debugHealthDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
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

	s.App.Get("/debug/health/details", s.HandleDebugHealthDetails)

	resp, err := s.App.Test(httptest.NewRequest(http.MethodGet, "/debug/health/details", nil))
	if err != nil {
		t.Fatalf("health details request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.StatusCode)
	}

	var payload debugHealthDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
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

func TestRequestLoggerMiddlewareBranches(t *testing.T) {
	app := fiber.New()
	app.Use(RequestLoggerMiddleware(obslogging.Logger{}))
	app.Get("/disabled", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/disabled", nil))
	if err != nil || resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("disabled logger request failed status=%d err=%v", resp.StatusCode, err)
	}

	provider, shutdown, err := obslogging.NewProvider(t.Context(), &appenv.Env{AppName: "miconsul", OTelOTLPEndpoint: "localhost:4317", OTelOTLPInsecure: true})
	if err != nil {
		t.Fatalf("new logging provider: %v", err)
	}
	defer func() { _ = shutdown() }()

	logger := obslogging.NewLogger(provider, "miconsul.test.requestlogger")
	app2 := fiber.New()
	app2.Use(RequestLoggerMiddleware(logger))
	app2.Get("/ok", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	app2.Get("/err", func(c fiber.Ctx) error { return errors.New("boom") })
	app2.Get("/public/file", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })

	resp, err = app2.Test(httptest.NewRequest(http.MethodGet, "/ok", nil))
	if err != nil || resp.StatusCode != fiber.StatusOK {
		t.Fatalf("enabled logger /ok request failed status=%d err=%v", resp.StatusCode, err)
	}

	resp, err = app2.Test(httptest.NewRequest(http.MethodGet, "/public/file", nil))
	if err != nil || resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("enabled logger /public request failed status=%d err=%v", resp.StatusCode, err)
	}

	resp, err = app2.Test(httptest.NewRequest(http.MethodGet, "/err", nil))
	if err != nil || resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("enabled logger /err request failed status=%d err=%v", resp.StatusCode, err)
	}
}
