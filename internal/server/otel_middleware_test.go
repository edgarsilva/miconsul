package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// TestOtelMiddlewareServesStaticFiles guards against gofiber/contrib#1734: otel
// middleware v1.2.4 wrapped static file body streams and panicked when fasthttp
// released the request, so static paths had to be skipped from tracing.
func TestOtelMiddlewareServesStaticFiles(t *testing.T) {
	setWorkingDirToRepoRoot(t)
	recorder := useSpanRecorder(t)

	s := &Server{
		Env: &appenv.Env{
			Environment:   appenv.EnvironmentProduction,
			AppProtocol:   "http",
			CookieSecret:  "0123456789abcdef0123456789abcdef",
			SessionDBPath: filepath.Join(t.TempDir(), "session.sqlite3"),
		},
		DB: &database.Database{DB: openTestDB(t)},
	}
	setupSessionStore(s)
	setupFiberApp(s)
	s.Get("/boom", func(c fiber.Ctx) error { return errors.New("boom") })

	tests := []struct {
		path           string
		acceptEncoding string
		wantStatus     int
	}{
		{path: "/public/404.css", wantStatus: fiber.StatusOK},
		{path: "/public/404.css", acceptEncoding: "gzip, br", wantStatus: fiber.StatusOK},
		{path: "/public/favicon.png", wantStatus: fiber.StatusOK},
		{path: "/.well-known/appspecific/com.chrome.devtools.json", wantStatus: fiber.StatusOK},
		{path: "/favicon.ico", wantStatus: fiber.StatusOK},
		{path: "/public/missing.css", wantStatus: fiber.StatusNotFound},
		{path: "/boom", wantStatus: fiber.StatusInternalServerError},
	}

	// Repeat so fasthttp serves from its file cache, where the panic surfaced.
	for range 3 {
		for _, tt := range tests {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}

			resp, err := s.App.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
			if err != nil {
				t.Fatalf("GET %s failed: %v", tt.path, err)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read GET %s body: %v", tt.path, err)
			}

			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("GET %s status = %d, want %d", tt.path, resp.StatusCode, tt.wantStatus)
			}

			if tt.wantStatus == fiber.StatusOK && len(body) == 0 {
				t.Fatalf("GET %s returned an empty body", tt.path)
			}
		}
	}

	var staticSpans, errorSpans int
	for _, span := range recorder.Ended() {
		for _, attr := range span.Attributes() {
			if attr.Key == "url.path" && attr.Value.AsString() == "/public/404.css" {
				staticSpans++
			}
		}
		if span.Name() == "GET /boom" && span.Status().Code == codes.Error {
			errorSpans++
		}
	}

	if staticSpans != 6 {
		t.Fatalf("expected 6 spans for /public/404.css, got %d", staticSpans)
	}

	if errorSpans != 3 {
		t.Fatalf("expected 3 error spans for GET /boom, got %d", errorSpans)
	}
}

// TestOtelMiddlewareRecordsHTTPMetrics guards the only HTTP metrics stream: the
// semconv http.server.request.duration histogram recorded by the otel middleware.
func TestOtelMiddlewareRecordsHTTPMetrics(t *testing.T) {
	setWorkingDirToRepoRoot(t)

	reader := sdkmetric.NewManualReader()

	s := &Server{
		MeterProvider: sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)),
		Env: &appenv.Env{
			Environment:   appenv.EnvironmentProduction,
			AppProtocol:   "http",
			CookieSecret:  "0123456789abcdef0123456789abcdef",
			SessionDBPath: filepath.Join(t.TempDir(), "session.sqlite3"),
		},
		DB: &database.Database{DB: openTestDB(t)},
	}
	setupSessionStore(s)
	setupFiberApp(s)
	s.Get("/patients/:id", func(c fiber.Ctx) error { return c.SendString("ok") })

	assertStatus(t, s, http.MethodGet, "/patients/123", fiber.StatusOK)
	assertStatus(t, s, http.MethodGet, "/patients/456", fiber.StatusOK)
	assertStatus(t, s, http.MethodGet, "/livez", fiber.StatusOK)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("collect metrics: %v", err)
	}

	counts := map[string]uint64{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "http.server.request.duration" {
				continue
			}
			hist, ok := m.Data.(metricdata.Histogram[float64])
			if !ok {
				t.Fatalf("http.server.request.duration data = %T, want float64 histogram", m.Data)
			}
			for _, dp := range hist.DataPoints {
				route, _ := dp.Attributes.Value("http.route")
				counts[route.AsString()] += dp.Count
			}
		}
	}

	if counts["/patients/:id"] != 2 {
		t.Fatalf("expected 2 requests recorded for /patients/:id, got %v", counts)
	}
	if _, ok := counts["/livez"]; ok {
		t.Fatalf("expected probe requests to be skipped, got %v", counts)
	}
}

func useSpanRecorder(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()

	recorder := tracetest.NewSpanRecorder()
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))
	t.Cleanup(func() { otel.SetTracerProvider(previous) })

	return recorder
}
