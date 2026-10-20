package server

import (
	"context"
	"sync"
	"testing"
	"time"

	"miconsul/internal/lib/appenv"

	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestSetupTelemetryUsesInjectedProviders(t *testing.T) {
	exporter := &recordingLogExporter{}
	s := &Server{
		Env:            &appenv.Env{AppName: "miconsul", OTelTracerServer: "miconsul.server"},
		TracerProvider: noop.NewTracerProvider(),
		LoggerProvider: sdklog.NewLoggerProvider(sdklog.WithProcessor(sdklog.NewSimpleProcessor(exporter))),
		StartedAt:      time.Now(),
		ReadyAt:        time.Now(),
	}

	setupTelemetry(s)
	if s.Tracer == nil {
		t.Fatal("expected tracer from the injected tracer provider")
	}

	emitStartupBootstrapLog(s)

	scopes := exporter.scopes()
	if len(scopes) != 1 || scopes[0] != "miconsul.http" {
		t.Fatalf("expected one record from scope miconsul.http, got %v", scopes)
	}
}

type recordingLogExporter struct {
	mu      sync.Mutex
	records []sdklog.Record
}

func (e *recordingLogExporter) Export(_ context.Context, records []sdklog.Record) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, r := range records {
		e.records = append(e.records, r.Clone())
	}
	return nil
}

func (e *recordingLogExporter) Shutdown(context.Context) error   { return nil }
func (e *recordingLogExporter) ForceFlush(context.Context) error { return nil }

func (e *recordingLogExporter) scopes() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	scopes := make([]string, 0, len(e.records))
	for _, r := range e.records {
		scopes = append(scopes, r.InstrumentationScope().Name)
	}
	return scopes
}
