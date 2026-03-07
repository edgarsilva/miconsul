package logging

import (
	"context"
	"fmt"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/observability"

	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
)

type Provider struct {
	provider *sdklog.LoggerProvider
}

type Logger struct {
	emitter otellog.Logger
}

func NewProvider(ctx context.Context, env *appenv.Env) (Provider, func() error, error) {
	if env == nil {
		return Provider{}, nil, fmt.Errorf("otel logs: environment config is nil")
	}

	if env.OTelOTLPEndpoint == "" {
		return Provider{}, func() error { return nil }, nil
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
		return Provider{}, nil, fmt.Errorf("otel logs: create otlp exporter: %w", err)
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
		return Provider{}, nil, fmt.Errorf("otel logs: create resource: %w", err)
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

	return Provider{provider: provider}, shutdown, nil
}

func NewLogger(provider Provider, scopeName string) Logger {
	if provider.provider == nil {
		return Logger{}
	}

	return Logger{emitter: provider.provider.Logger(scopeName)}
}

func (l Logger) Enabled() bool {
	return l.emitter != nil
}

func (l Logger) Emit(ctx context.Context, record otellog.Record) {
	if l.emitter == nil {
		return
	}

	l.emitter.Emit(ctx, record)
}
