package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"miconsul/internal/lib/appenv"
	obslogging "miconsul/internal/observability/logging"

	"github.com/gofiber/fiber/v3"
	fiberrecover "github.com/gofiber/fiber/v3/middleware/recover"
	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	lognoop "go.opentelemetry.io/otel/log/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

func TestRequestLoggerMiddlewareBranches(t *testing.T) {
	app := fiber.New()
	app.Use(RequestLoggerMiddleware(lognoop.Logger{}))
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

	logger := provider.Logger("miconsul.test.requestlogger")
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

type recordCapture struct {
	records []sdklog.Record
}

func (r *recordCapture) OnEmit(_ context.Context, rec *sdklog.Record) error {
	r.records = append(r.records, rec.Clone())
	return nil
}
func (r *recordCapture) Enabled(context.Context, sdklog.EnabledParameters) bool { return true }
func (r *recordCapture) Shutdown(context.Context) error                         { return nil }
func (r *recordCapture) ForceFlush(context.Context) error                       { return nil }

// TestRequestLoggerMiddlewareLogsPanic checks that a handler panic still emits
// an http_request error record, and that the panic reaches the recover middleware.
func TestRequestLoggerMiddlewareLogsPanic(t *testing.T) {
	capture := &recordCapture{}
	provider := sdklog.NewLoggerProvider(sdklog.WithProcessor(capture))
	defer func() { _ = provider.Shutdown(t.Context()) }()

	app := fiber.New()
	app.Use(fiberrecover.New())
	app.Use(RequestLoggerMiddleware(provider.Logger("miconsul.test.requestlogger")))
	app.Get("/panic", func(c fiber.Ctx) error { panic("boom") })

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/panic", nil))
	if err != nil || resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("GET /panic status=%d err=%v, want 500", resp.StatusCode, err)
	}
	if len(capture.records) != 1 {
		t.Fatalf("records = %d, want 1", len(capture.records))
	}

	rec := capture.records[0]
	if rec.EventName() != "http_request" || rec.Severity() != otellog.SeverityError {
		t.Fatalf("event=%q severity=%v, want http_request ERROR", rec.EventName(), rec.Severity())
	}
	attrs := map[string]attribute.Value{}
	rec.WalkAttributes(func(kv attribute.KeyValue) bool { attrs[string(kv.Key)] = kv.Value; return true })
	if got := attrs["status"].AsInt64(); got != fiber.StatusInternalServerError {
		t.Fatalf("status attr = %d, want 500", got)
	}
	if got := attrs["error"].AsString(); !strings.Contains(got, "boom") {
		t.Fatalf("error attr = %q, want it to contain the panic value", got)
	}
}
