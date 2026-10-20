package tracing

import (
	"context"
	"fmt"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/observability"

	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// NewProvider returns the tracer provider and its shutdown func. It returns a
// no-op provider when env has no OTLP endpoint, and sets the global tracer
// provider and propagators otherwise.
func NewProvider(ctx context.Context, env *appenv.Env) (trace.TracerProvider, func() error, error) {
	if env == nil {
		return nil, nil, fmt.Errorf("otel tracing: environment config is nil")
	}

	if env.OTelOTLPEndpoint == "" {
		return noop.NewTracerProvider(), func() error { return nil }, nil
	}

	serviceName := env.OTelServiceName
	if serviceName == "" {
		serviceName = env.AppName
	}

	otlpOpts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(env.OTelOTLPEndpoint)}
	if env.OTelOTLPInsecure || observability.IsInternalOTLPEndpoint(env.OTelOTLPEndpoint) {
		otlpOpts = append(otlpOpts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, otlpOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("otel tracing: create otlp exporter: %w", err)
	}

	res, err := resource.New(
		ctx,
		resource.WithHost(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(env.AppVersion),
			semconv.DeploymentEnvironmentKey.String(string(env.Environment)),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("otel tracing: create resource: %w", err)
	}

	bsp := sdktrace.NewBatchSpanProcessor(exporter,
		sdktrace.WithMaxQueueSize(10_000),
		sdktrace.WithMaxExportBatchSize(10_000),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithIDGenerator(xray.NewIDGenerator()),
		sdktrace.WithSpanProcessor(bsp),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	shutdown := func() error {
		if err := tp.Shutdown(context.Background()); err != nil {
			return fmt.Errorf("otel tracing: shutdown tracer provider: %w", err)
		}
		return nil
	}

	return tp, shutdown, nil
}
