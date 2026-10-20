package tracing

import (
	"testing"

	"miconsul/internal/lib/appenv"

	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewProviderNilEnv(t *testing.T) {
	if _, _, err := NewProvider(t.Context(), nil); err == nil {
		t.Fatalf("expected NewProvider to fail with nil env")
	}
}

func TestNewProviderNoopWithoutOTLP(t *testing.T) {
	env := &appenv.Env{AppName: "miconsul", Environment: appenv.EnvironmentDevelopment}
	provider, shutdown, err := NewProvider(t.Context(), env)
	if err != nil {
		t.Fatalf("expected no-endpoint provider to succeed: %v", err)
	}
	if _, ok := provider.(noop.TracerProvider); !ok {
		t.Fatalf("expected noop tracer provider, got %T", provider)
	}
	if err := shutdown(); err != nil {
		t.Fatalf("expected noop shutdown: %v", err)
	}
}

func TestNewProviderOTLPBranch(t *testing.T) {
	env := &appenv.Env{AppName: "miconsul", OTelOTLPEndpoint: "localhost:4317", OTelOTLPInsecure: true}
	provider, shutdown, err := NewProvider(t.Context(), env)
	if err != nil {
		t.Fatalf("expected otlp provider setup to succeed: %v", err)
	}
	if provider == nil {
		t.Fatalf("expected non-nil otlp tracer provider")
	}
	_ = shutdown()
}
