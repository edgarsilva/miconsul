package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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

func TestReadinessProbeFailsWhenDatabaseClosed(t *testing.T) {
	srv := readyServer(t)
	sqlDB, err := srv.DB.SQLDB()
	if err != nil {
		t.Fatalf("get sql db handle: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	probe := readinessProbe(srv)
	app := fiber.New()
	app.Get("/ready", func(c fiber.Ctx) error {
		if probe(c) {
			return c.SendStatus(http.StatusNoContent)
		}
		return c.SendStatus(http.StatusServiceUnavailable)
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("readiness request failed: %v", err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected readiness failure, got %d", resp.StatusCode)
	}
}

func TestReadinessProbeFailsWhenDatabaseIsLockedBeyondProbeTimeout(t *testing.T) {
	srv := readyServerWithBusyTimeout(t)
	sqlDB, err := srv.DB.SQLDB()
	if err != nil {
		t.Fatalf("get sql db handle: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	lockConn, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatalf("acquire lock connection: %v", err)
	}
	t.Cleanup(func() {
		_ = lockConn.Close()
	})

	if _, err := lockConn.ExecContext(context.Background(), "BEGIN EXCLUSIVE"); err != nil {
		t.Fatalf("begin exclusive lock: %v", err)
	}
	t.Cleanup(func() {
		_, _ = lockConn.ExecContext(context.Background(), "ROLLBACK")
	})

	probe := readinessProbe(srv)
	app := fiber.New()
	app.Get("/ready", func(c fiber.Ctx) error {
		if probe(c) {
			return c.SendStatus(http.StatusNoContent)
		}
		return c.SendStatus(http.StatusServiceUnavailable)
	})

	start := time.Now()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
	if err != nil {
		t.Fatalf("readiness request failed: %v", err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected readiness failure under lock, got %d", resp.StatusCode)
	}
	if elapsed := time.Since(start); elapsed < healthProbeTimeout {
		t.Fatalf("expected lock wait to exceed probe timeout, got %v", elapsed)
	}
	if elapsed := time.Since(start); elapsed > 6*time.Second {
		t.Fatalf("expected lock wait to stay below busy timeout window, got %v", elapsed)
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

func readyServerWithBusyTimeout(t *testing.T) *Server {
	t.Helper()

	path := filepath.Join(t.TempDir(), "server_probe_busy.sqlite")
	dsn := fmt.Sprintf("%s?_busy_timeout=5000", path)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite with busy timeout: %v", err)
	}

	return &Server{DB: &database.Database{DB: db}}
}
