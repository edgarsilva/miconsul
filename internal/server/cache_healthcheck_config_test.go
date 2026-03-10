package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type cacheStub struct {
	readErr  error
	writeErr error
	wroteKey string
	readKey  string
}

func (c *cacheStub) Read(key string, _ *[]byte) error {
	c.readKey = key
	return c.readErr
}

func (c *cacheStub) Write(key string, _ *[]byte, _ time.Duration) error {
	c.wroteKey = key
	return c.writeErr
}

func TestCacheReadWrite(t *testing.T) {
	s := &Server{}
	payload := []byte("v")

	if err := s.CacheWrite("k", &payload, time.Second); err != nil {
		t.Fatalf("expected nil cache write to be no-op: %v", err)
	}
	if err := s.CacheRead("k", &payload); err != nil {
		t.Fatalf("expected nil cache read to be no-op: %v", err)
	}

	stub := &cacheStub{}
	s.Cache = stub
	if err := s.CacheWrite("write-key", &payload, time.Second); err != nil {
		t.Fatalf("unexpected cache write error: %v", err)
	}
	if stub.wroteKey != "write-key" {
		t.Fatalf("expected write key to be captured, got %q", stub.wroteKey)
	}

	if err := s.CacheRead("read-key", &payload); err != nil {
		t.Fatalf("unexpected cache read error: %v", err)
	}
	if stub.readKey != "read-key" {
		t.Fatalf("expected read key to be captured, got %q", stub.readKey)
	}

	stub.writeErr = errors.New("write failed")
	if err := s.CacheWrite("k2", &payload, time.Second); err == nil {
		t.Fatalf("expected cache write error")
	}
}

func TestStaticConfigHelpers(t *testing.T) {
	dev := &appenv.Env{Environment: appenv.EnvironmentDevelopment}
	prod := &appenv.Env{Environment: appenv.EnvironmentProduction}

	if got := staticCacheDuration(dev); got != 0 {
		t.Fatalf("expected dev static cache duration 0, got %v", got)
	}
	if got := staticMaxAge(dev); got != 0 {
		t.Fatalf("expected dev static max age 0, got %d", got)
	}
	if got := staticCacheDuration(prod); got != 300*time.Second {
		t.Fatalf("expected prod static cache duration 300s, got %v", got)
	}
	if got := staticMaxAge(prod); got != 3600 {
		t.Fatalf("expected prod static max age 3600, got %d", got)
	}
}

func TestSessionConfigDefaults(t *testing.T) {
	cfg := sessionConfig("")
	if cfg.Database == "" {
		t.Fatalf("expected default session db path")
	}
	if cfg.Table != "fiber_storage" {
		t.Fatalf("expected default session table name, got %q", cfg.Table)
	}
}

func TestLivenessReadinessStartupProbes(t *testing.T) {
	if livenessProbe(nil) == nil {
		t.Fatalf("expected liveness probe function")
	}

	app := fiber.New()
	app.Get("/probe", func(c fiber.Ctx) error {
		if livenessProbe(&Server{App: app})(c) {
			return c.SendStatus(http.StatusNoContent)
		}
		return c.SendStatus(http.StatusServiceUnavailable)
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("liveness request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected liveness success, got %d", resp.StatusCode)
	}

	readySrv := readyServer(t)
	readyProbe := readinessProbe(readySrv)
	startup := startupProbe(&Server{
		DB:                readySrv.DB,
		StartedAt:         time.Now().Add(-3 * time.Second),
		ReadyAt:           time.Now().Add(-2 * time.Second),
		BootstrapDuration: time.Second,
	})

	app = fiber.New()
	app.Get("/ready", func(c fiber.Ctx) error {
		if readyProbe(c) {
			return c.SendStatus(http.StatusNoContent)
		}
		return c.SendStatus(http.StatusServiceUnavailable)
	})
	app.Get("/startup", func(c fiber.Ctx) error {
		if startup(c) {
			return c.SendStatus(http.StatusNoContent)
		}
		return c.SendStatus(http.StatusServiceUnavailable)
	})

	req = httptest.NewRequest(http.MethodGet, "/ready", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("readiness request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected readiness success, got %d", resp.StatusCode)
	}

	req = httptest.NewRequest(http.MethodGet, "/startup", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("startup request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected startup success, got %d", resp.StatusCode)
	}
}

func readyServer(t *testing.T) *Server {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:server_probe?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}

	return &Server{DB: &database.Database{DB: db}}
}
