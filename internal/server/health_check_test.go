package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"miconsul/internal/database"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

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
