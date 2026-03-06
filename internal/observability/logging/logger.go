package logging

import (
	"context"
	"fmt"
	"time"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/observability"

	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
)

type HTTPEvent struct {
	Route      string
	Method     string
	Status     int
	DurationMS float64
	TraceID    string
	Err        error
}

type Logger struct {
	emitter otellog.Logger
}

func New(ctx context.Context, env *appenv.Env) (Logger, func() error, error) {
	if env == nil {
		return Logger{}, nil, fmt.Errorf("otel logs: environment config is nil")
	}

	if env.OTelOTLPEndpoint == "" {
		return Logger{}, func() error { return nil }, nil
	}

	serviceName := env.OTelServiceName
	if serviceName == "" {
		serviceName = env.AppName
	}

	otlpOpts := []otlploggrpc.Option{otlploggrpc.WithEndpoint(env.OTelOTLPEndpoint)}
	if env.OTelOTLPInsecure || observability.IsInternalOTLPEndpoint(env.OTelOTLPEndpoint) {
		otlpOpts = append(otlpOpts, otlploggrpc.WithInsecure())
	}

	exporter, err := otlploggrpc.New(ctx, otlpOpts...)
	if err != nil {
		return Logger{}, nil, fmt.Errorf("otel logs: create otlp exporter: %w", err)
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
		return Logger{}, nil, fmt.Errorf("otel logs: create resource: %w", err)
	}

	processor := sdklog.NewBatchProcessor(exporter)
	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(processor),
		sdklog.WithResource(res),
	)

	global.SetLoggerProvider(provider)

	shutdown := func() error {
		if err := provider.Shutdown(context.Background()); err != nil {
			return fmt.Errorf("otel logs: shutdown logger provider: %w", err)
		}

		return nil
	}

	return Logger{emitter: provider.Logger(env.AppName + ".http")}, shutdown, nil
}

func (l Logger) Enabled() bool {
	return l.emitter != nil
}

func (l Logger) EmitHTTP(ctx context.Context, event HTTPEvent) {
	if l.emitter == nil {
		return
	}

	rec := otellog.Record{}
	rec.SetTimestamp(time.Now())
	rec.SetObservedTimestamp(time.Now())
	rec.SetEventName("http_request")
	rec.SetBody(otellog.StringValue("http_request"))

	if event.Status >= 500 {
		rec.SetSeverity(otellog.SeverityError)
		rec.SetSeverityText("ERROR")
	} else {
		rec.SetSeverity(otellog.SeverityInfo)
		rec.SetSeverityText("INFO")
	}

	attrs := []otellog.KeyValue{
		otellog.String("event", "http_request"),
		otellog.String("route", event.Route),
		otellog.String("method", event.Method),
		otellog.Int("status", event.Status),
		otellog.Float64("duration_ms", event.DurationMS),
	}
	if event.TraceID != "" {
		attrs = append(attrs, otellog.String("trace_id", event.TraceID))
	}
	if event.Err != nil {
		rec.SetErr(event.Err)
		attrs = append(attrs, otellog.String("error", event.Err.Error()))
	}
	rec.AddAttributes(attrs...)

	l.emitter.Emit(ctx, rec)
}
