package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"miconsul/internal/lib/appenv"

	"github.com/gofiber/fiber/v3"
)

func TestLimiterStaticHelmetConfig(t *testing.T) {
	lcfg := limiterConfig()
	if lcfg.Max != 100 {
		t.Fatalf("expected limiter max 100, got %d", lcfg.Max)
	}

	dev := &appenv.Env{Environment: appenv.EnvironmentDevelopment}
	pcfg := staticConfig(dev)
	if pcfg.CacheDuration != 0 || pcfg.MaxAge != 0 {
		t.Fatalf("expected dev static cache disabled, got duration=%v maxage=%d", pcfg.CacheDuration, pcfg.MaxAge)
	}

	hcfg := helmetConfig()
	if hcfg.ContentSecurityPolicy == "" || hcfg.XFrameOptions == "" {
		t.Fatalf("expected helmet config to set security headers")
	}
}

func TestSendErrorPageAndFiberErrorHandler(t *testing.T) {
	tmp := t.TempDir()
	publicDir := filepath.Join(tmp, "public")
	if err := os.MkdirAll(publicDir, 0o755); err != nil {
		t.Fatalf("mkdir public dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(publicDir, "404.html"), []byte("<h1>Not Found Page</h1>"), 0o644); err != nil {
		t.Fatalf("write 404 page: %v", err)
	}
	if err := os.WriteFile(filepath.Join(publicDir, "500.html"), []byte("<h1>Internal Error Page</h1>"), 0o644); err != nil {
		t.Fatalf("write 500 page: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir failed: %v", err)
	}

	errorPagesOnce = sync.Once{}
	errorPages = map[int]string{}

	app := fiber.New(fiber.Config{ErrorHandler: fiberAppErrorHandler})
	app.Get("/nf", func(c fiber.Ctx) error {
		return fiber.NewError(fiber.StatusNotFound, "missing")
	})
	app.Get("/boom", func(c fiber.Ctx) error {
		return errors.New("boom")
	})

	resp404, err := app.Test(httptest.NewRequest(http.MethodGet, "/nf", nil))
	if err != nil {
		t.Fatalf("404 request failed: %v", err)
	}
	if resp404.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 status, got %d", resp404.StatusCode)
	}

	resp500, err := app.Test(httptest.NewRequest(http.MethodGet, "/boom", nil))
	if err != nil {
		t.Fatalf("500 request failed: %v", err)
	}
	if resp500.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500 status, got %d", resp500.StatusCode)
	}
}
