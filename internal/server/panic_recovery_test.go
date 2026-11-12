package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// TestHandlerPanicIsRecorded checks that a panic in a handler becomes a 500 and
// is recorded on the request span with a stack trace, so it can be found and fixed.
func TestHandlerPanicIsRecorded(t *testing.T) {
	setWorkingDirToRepoRoot(t)
	recorder := tracetest.NewSpanRecorder()

	s := &Server{
		TracerProvider: sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)),
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
	s.Get("/panic", func(c fiber.Ctx) error {
		panic("boom")
	})

	resp, err := s.App.Test(httptest.NewRequest(http.MethodGet, "/panic", nil), fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("GET /panic failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}

	for _, span := range recorder.Ended() {
		if span.Name() != "GET /panic" {
			continue
		}
		if span.Status().Code != codes.Error {
			t.Fatalf("span status = %v, want Error", span.Status().Code)
		}
		for _, event := range span.Events() {
			if event.Name != "exception" {
				continue
			}
			var hasPanic, hasStack bool
			for _, attr := range event.Attributes {
				value := attr.Value.AsString()
				switch string(attr.Key) {
				case "exception.message":
					hasPanic = strings.Contains(value, "boom")
				case "exception.stacktrace":
					hasStack = value != ""
				}
			}
			if !hasPanic || !hasStack {
				t.Fatalf("exception event needs the panic message and a stack trace: %+v", event.Attributes)
			}
			return
		}
		t.Fatalf("span GET /panic has no exception event: %+v", span.Events())
	}
	t.Fatalf("no span named GET /panic was recorded")
}
