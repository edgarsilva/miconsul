package metrics

import (
	"context"
	"fmt"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/observability"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// NewProvider returns the meter provider and its shutdown func. It returns a
// no-op provider when env has no OTLP endpoint, and sets the global meter
// provider otherwise.
func NewProvider(ctx context.Context, env *appenv.Env) (metric.MeterProvider, func() error, error) {
	if env == nil {
		return nil, nil, fmt.Errorf("otel metrics: environment config is nil")
	}

	if env.OTelOTLPEndpoint == "" {
		return noop.NewMeterProvider(), func() error { return nil }, nil
	}

	endpoint := env.OTelOTLPEndpoint
	serviceName := env.OTelServiceName
	if serviceName == "" {
		serviceName = env.AppName
	}

	otlpOpts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(endpoint)}
	if env.OTelOTLPInsecure || observability.IsInternalOTLPEndpoint(endpoint) {
		otlpOpts = append(otlpOpts, otlpmetricgrpc.WithInsecure())
	}

	exporter, err := otlpmetricgrpc.New(ctx, otlpOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("otel metrics: create otlp exporter: %w", err)
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
		return nil, nil, fmt.Errorf("otel metrics: create resource: %w", err)
	}

	reader := sdkmetric.NewPeriodicReader(exporter)
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithResource(res),
	)

	otel.SetMeterProvider(mp)

	shutdown := func() error {
		if err := mp.Shutdown(context.Background()); err != nil {
			return fmt.Errorf("otel metrics: shutdown meter provider: %w", err)
		}
		return nil
	}

	return mp, shutdown, nil
}
