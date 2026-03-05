package telemetry

import (
	"context"
	"fmt"

	"miconsul/internal/lib/appenv"

	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"

	//"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
)

func NewTracer(ctx context.Context, name string, env *appenv.Env) (tracer trace.Tracer, shutdownFn func() error, err error) {
	if env == nil {
		return nil, nil, fmt.Errorf("otel: environment config is nil")
	}

	if appenv.IsDevelopment(env.Environment) {
		tracer, shutdownFn, err = NewDevTracer(ctx, name)
		return tracer, shutdownFn, err
	}

	tracer, shutdownFn, err = NewUptraceTracer(ctx, name, env)
	return tracer, shutdownFn, err
}

func NewDevTracer(ctx context.Context, name string) (tracer trace.Tracer, shutdownFn func() error, err error) {
	tracer = otel.Tracer(name)
	return tracer, func() error {
		return nil
	}, nil
}

func NewStdoutTracer(ctx context.Context, name string, env *appenv.Env) (tracer trace.Tracer, shutdownFn func() error, err error) {
	if env == nil {
		return nil, nil, fmt.Errorf("otel: environment config is nil")
	}

	serviceName := env.OTelServiceName
	if serviceName == "" {
		serviceName = env.AppName
	}

	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, nil, fmt.Errorf("otel: create stdout exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(serviceName),
			)),
	)

	otel.SetTracerProvider(tp)

	textPropagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)

	otel.SetTextMapPropagator(textPropagator)

	tracer = tp.Tracer(name)
	return tracer, func() error {
		if err := tp.Shutdown(context.Background()); err != nil {
			return fmt.Errorf("otel: shutdown tracer provider: %w", err)
		}

		return nil
	}, nil
}

func NewUptraceTracer(ctx context.Context, name string, env *appenv.Env) (tracer trace.Tracer, shutdownFn func() error, err error) {
	if env == nil {
		return nil, nil, fmt.Errorf("otel: environment config is nil")
	}

	var (
		dsn         = env.UptraceDSN
		endpoint    = env.UptraceEndpoint
		appVersion  = env.AppVersion
		serviceName = env.OTelServiceName
	)

	if serviceName == "" {
		serviceName = env.AppName
	}

	if dsn == "" || endpoint == "" {
		return nil, nil, fmt.Errorf("otel: uptrace dsn or endpoint missing")
	}

	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithHeaders(map[string]string{
			"uptrace-dsn": dsn,
		}),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("otel: create uptrace exporter: %w", err)
	}

	resource, err := resource.New(
		ctx,
		resource.WithHost(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(appVersion),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("otel: create resource: %w", err)
	}

	bsp := sdktrace.NewBatchSpanProcessor(exporter,
		sdktrace.WithMaxQueueSize(10_000),
		sdktrace.WithMaxExportBatchSize(10_000),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(resource),
		sdktrace.WithIDGenerator(xray.NewIDGenerator()),
	)

	tp.RegisterSpanProcessor(bsp)
	otel.SetTracerProvider(tp)

	textPropagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)

	otel.SetTextMapPropagator(textPropagator)

	tracer = tp.Tracer(name)
	return tracer, func() error {
		if err := tp.Shutdown(context.Background()); err != nil {
			return fmt.Errorf("otel: shutdown tracer provider: %w", err)
		}

		return nil
	}, nil
}
