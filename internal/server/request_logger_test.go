package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"miconsul/internal/lib/appenv"
	obslogging "miconsul/internal/observability/logging"

	"github.com/gofiber/fiber/v3"
)

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
