package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

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

// TestHandlerPanicIsRecorded checks that a panic in a handler becomes a 500 and
// is recorded on the request span with a stack trace, so it can be found and fixed.
func TestHandlerPanicIsRecorded(t *testing.T) {
	setWorkingDirToRepoRoot(t)
	recorder := tracetest.NewSpanRecorder()
	capture := &recordCapture{}
	logs := sdklog.NewLoggerProvider(sdklog.WithProcessor(capture))
	defer func() { _ = logs.Shutdown(t.Context()) }()

	s := &Server{
		Logger:         logs.Logger("miconsul.test.panic"),
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

	checkPanicLogRecord(t, capture)

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

// checkPanicLogRecord checks the request_panic record: error severity, the
// route, the panic value and a stack trace.
func checkPanicLogRecord(t *testing.T, capture *recordCapture) {
	t.Helper()

	var found *sdklog.Record
	for i := range capture.records {
		if capture.records[i].EventName() == "request_panic" {
			found = &capture.records[i]
		}
	}
	if found == nil {
		t.Fatalf("no request_panic log record was emitted: %d records", len(capture.records))
	}
	if found.Severity() != otellog.SeverityError {
		t.Fatalf("severity = %v, want ERROR", found.Severity())
	}

	attrs := map[string]attribute.Value{}
	found.WalkAttributes(func(kv attribute.KeyValue) bool { attrs[string(kv.Key)] = kv.Value; return true })
	if got := attrs["route"].AsString(); got != "/panic" {
		t.Fatalf("route = %q, want /panic", got)
	}
	if got := attrs["error"].AsString(); !strings.Contains(got, "boom") {
		t.Fatalf("error = %q, want it to contain the panic value", got)
	}
	if got := attrs["exception.stacktrace"].AsString(); got == "" {
		t.Fatalf("exception.stacktrace is empty")
	}
}
