package metrics

import (
	"testing"

	"miconsul/internal/lib/appenv"
)

func TestNewProviderGuardsAndNoEndpointPath(t *testing.T) {
	if _, _, err := NewProvider(t.Context(), nil); err == nil {
		t.Fatalf("expected nil env error")
	}

	env := &appenv.Env{AppName: "miconsul"}
	provider, shutdown, err := NewProvider(t.Context(), env)
	if err != nil {
		t.Fatalf("expected no-endpoint meter provider to succeed: %v", err)
	}
	if provider == nil {
		t.Fatalf("expected non-nil meter provider")
	}
	if shutdown == nil {
		t.Fatalf("expected shutdown callback")
	}
	if err := shutdown(); err != nil {
		t.Fatalf("expected noop shutdown without otlp endpoint: %v", err)
	}
}

func TestNewProviderOTLPBranch(t *testing.T) {
	env := &appenv.Env{AppName: "miconsul", OTelOTLPEndpoint: "localhost:4317", OTelOTLPInsecure: true}

	provider, shutdown, err := NewProvider(t.Context(), env)
	if err != nil {
		t.Fatalf("expected otlp meter provider setup to succeed: %v", err)
	}
	if provider == nil {
		t.Fatalf("expected non-nil otlp meter provider")
	}
	if shutdown != nil {
		_ = shutdown()
	}
}
