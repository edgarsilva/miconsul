package metrics

import (
	"testing"

	"miconsul/internal/lib/appenv"
)

func TestNewMeterProviderGuardsAndNoEndpointPath(t *testing.T) {
	if _, _, err := NewMeterProvider(t.Context(), nil); err == nil {
		t.Fatalf("expected nil env error")
	}

	env := &appenv.Env{AppName: "miconsul"}
	meter, shutdown, err := NewMeterProvider(t.Context(), env)
	if err != nil {
		t.Fatalf("expected no-endpoint meter provider to succeed: %v", err)
	}
	if meter == nil {
		t.Fatalf("expected non-nil meter")
	}
	if shutdown == nil {
		t.Fatalf("expected shutdown callback")
	}
	if err := shutdown(); err != nil {
		t.Fatalf("expected noop shutdown without otlp endpoint: %v", err)
	}
}
