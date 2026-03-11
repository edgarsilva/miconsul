package logging

import (
	"testing"

	"miconsul/internal/lib/appenv"

	otellog "go.opentelemetry.io/otel/log"
)

func TestLoggerProviderGuardsAndNoEndpointPath(t *testing.T) {
	if _, _, err := NewProvider(t.Context(), nil); err == nil {
		t.Fatalf("expected nil env error")
	}

	env := &appenv.Env{AppName: "miconsul"}
	provider, shutdown, err := NewProvider(t.Context(), env)
	if err != nil {
		t.Fatalf("expected no-endpoint provider to succeed: %v", err)
	}
	if shutdown == nil {
		t.Fatalf("expected shutdown callback")
	}
	if err := shutdown(); err != nil {
		t.Fatalf("expected noop shutdown without endpoint: %v", err)
	}

	logger := NewLogger(provider, "miconsul.test")
	if logger.Enabled() {
		t.Fatalf("expected disabled logger when provider is zero")
	}
	logger.Emit(t.Context(), otellog.Record{})
}

func TestLoggerProviderOTLPBranch(t *testing.T) {
	env := &appenv.Env{
		AppName:          "miconsul",
		OTelOTLPEndpoint: "localhost:4317",
		OTelOTLPInsecure: true,
	}

	provider, shutdown, err := NewProvider(t.Context(), env)
	if err != nil {
		t.Fatalf("expected otlp provider setup to succeed: %v", err)
	}
	logger := NewLogger(provider, "miconsul.test.otlp")
	if !logger.Enabled() {
		t.Fatalf("expected logger to be enabled with otlp provider")
	}
	if shutdown != nil {
		_ = shutdown()
	}
}
