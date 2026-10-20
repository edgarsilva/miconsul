package logging

import (
	"testing"

	"miconsul/internal/lib/appenv"

	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
)

func TestNewProviderNoopWithoutOTLP(t *testing.T) {
	if _, _, err := NewProvider(t.Context(), nil); err == nil {
		t.Fatalf("expected nil env error")
	}

	provider, shutdown, err := NewProvider(t.Context(), &appenv.Env{AppName: "miconsul"})
	if err != nil {
		t.Fatalf("expected no-endpoint provider to succeed: %v", err)
	}
	if _, ok := provider.(noop.LoggerProvider); !ok {
		t.Fatalf("expected noop logger provider, got %T", provider)
	}
	if err := shutdown(); err != nil {
		t.Fatalf("expected noop shutdown without endpoint: %v", err)
	}
}

func TestNewProviderOTLPBranch(t *testing.T) {
	env := &appenv.Env{
		AppName:          "miconsul",
		OTelOTLPEndpoint: "localhost:4317",
		OTelOTLPInsecure: true,
	}

	provider, shutdown, err := NewProvider(t.Context(), env)
	if err != nil {
		t.Fatalf("expected otlp provider setup to succeed: %v", err)
	}
	defer func() { _ = shutdown() }()

	logger := provider.Logger("miconsul.test.otlp")
	if !logger.Enabled(t.Context(), otellog.EnabledParameters{}) {
		t.Fatalf("expected logger to be enabled with otlp provider")
	}
}
