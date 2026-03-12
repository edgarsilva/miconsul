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

func TestMetricsProviderAndFactoryBranches(t *testing.T) {
	env := &appenv.Env{AppName: "miconsul"}
	metrics, shutdown, err := New(t.Context(), env)
	if err != nil {
		t.Fatalf("expected metrics factory to succeed without endpoint: %v", err)
	}
	if metrics.HTTPDuration == nil || metrics.HTTPRequests == nil || metrics.PromHTTPDuration == nil || metrics.PromHTTPRequests == nil {
		t.Fatalf("expected all metrics instruments to be initialized")
	}
	if err := shutdown(); err != nil {
		t.Fatalf("expected metrics shutdown to succeed: %v", err)
	}
}

func TestMetricsProviderOTLPBranch(t *testing.T) {
	env := &appenv.Env{AppName: "miconsul", OTelOTLPEndpoint: "localhost:4317", OTelOTLPInsecure: true}

	meter, shutdown, err := NewMeterProvider(t.Context(), env)
	if err != nil {
		t.Fatalf("expected otlp meter provider setup to succeed: %v", err)
	}
	if meter == nil {
		t.Fatalf("expected non-nil meter from otlp provider")
	}
	if shutdown != nil {
		_ = shutdown()
	}
}
