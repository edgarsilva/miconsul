package logging

import (
	"context"
	"fmt"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/observability"

	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/log/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
)

// NewProvider returns the logger provider and its shutdown func. It returns a
// no-op provider when env has no OTLP endpoint, and sets the global logger
// provider otherwise.
func NewProvider(ctx context.Context, env *appenv.Env) (otellog.LoggerProvider, func() error, error) {
	if env == nil {
		return nil, nil, fmt.Errorf("otel logs: environment config is nil")
	}

	if env.OTelOTLPEndpoint == "" {
		return noop.NewLoggerProvider(), func() error { return nil }, nil
	}

	otlpOpts := []otlploggrpc.Option{otlploggrpc.WithEndpoint(env.OTelOTLPEndpoint)}
	if env.OTelOTLPInsecure {
		otlpOpts = append(otlpOpts, otlploggrpc.WithInsecure())
	}

	exporter, err := otlploggrpc.New(ctx, otlpOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("otel logs: create otlp exporter: %w", err)
	}

	res, err := observability.NewResource(ctx, env)
	if err != nil {
		return nil, nil, fmt.Errorf("otel logs: %w", err)
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

	return provider, shutdown, nil
}
