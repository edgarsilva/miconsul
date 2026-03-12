package tracing

import (
	"strings"
	"testing"

	"miconsul/internal/lib/appenv"
)

func TestTracingNilEnvGuards(t *testing.T) {
	if _, _, err := NewTracer(t.Context(), "trace", nil); err == nil {
		t.Fatalf("expected NewTracer to fail with nil env")
	}
	if _, _, err := NewStdoutTracer(t.Context(), "trace", nil); err == nil {
		t.Fatalf("expected NewStdoutTracer to fail with nil env")
	}
	if _, _, err := NewOTLPTracer(t.Context(), "trace", nil); err == nil {
		t.Fatalf("expected NewOTLPTracer to fail with nil env")
	}
}

func TestNewTracerFallsBackWithoutOTLP(t *testing.T) {
	env := &appenv.Env{AppName: "miconsul"}
	tracer, shutdown, err := NewTracer(t.Context(), "trace", env)
	if err != nil {
		t.Fatalf("new tracer fallback should succeed: %v", err)
	}
	if tracer == nil {
		t.Fatalf("expected non-nil tracer fallback")
	}
	if shutdown == nil {
		t.Fatalf("expected shutdown callback")
	}
	if err := shutdown(); err != nil {
		t.Fatalf("shutdown fallback should be noop: %v", err)
	}
}

func TestStdoutTracerAndOTLPErrorBranches(t *testing.T) {
	env := &appenv.Env{AppName: "miconsul"}
	tracer, shutdown, err := NewStdoutTracer(t.Context(), "trace", env)
	if err != nil {
		t.Fatalf("expected stdout tracer to succeed: %v", err)
	}
	if tracer == nil {
		t.Fatalf("expected non-nil stdout tracer")
	}
	if err := shutdown(); err != nil {
		t.Fatalf("expected stdout tracer shutdown to succeed: %v", err)
	}

	_, _, err = NewOTLPTracer(t.Context(), "trace", &appenv.Env{AppName: "miconsul"})
	if err == nil {
		t.Fatalf("expected missing endpoint error")
	}
	if !strings.Contains(err.Error(), "otlp endpoint missing") {
		t.Fatalf("expected missing endpoint message, got %v", err)
	}
}

func TestNewTracerOTLPBranchWrapsExporterErrors(t *testing.T) {
	env := &appenv.Env{AppName: "miconsul", OTelOTLPEndpoint: "localhost:4317", OTelOTLPInsecure: true}
	tracer, shutdown, err := NewTracer(t.Context(), "trace", env)
	if err != nil {
		t.Fatalf("expected otlp tracer setup to succeed: %v", err)
	}
	if tracer == nil {
		t.Fatalf("expected non-nil tracer")
	}
	if shutdown != nil {
		_ = shutdown()
	}
}
