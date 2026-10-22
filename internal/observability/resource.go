// Package observability holds what the logging, metrics and tracing providers share.
package observability

import (
	"context"
	"fmt"

	"miconsul/internal/lib/appenv"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// NewResource returns the OTel resource that describes this service to every signal.
func NewResource(ctx context.Context, env *appenv.Env) (*resource.Resource, error) {
	serviceName := env.OTelServiceName
	if serviceName == "" {
		serviceName = env.AppName
	}

	res, err := resource.New(
		ctx,
		resource.WithHost(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(env.AppVersion),
			semconv.DeploymentEnvironmentNameKey.String(string(env.Environment)),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create otel resource: %w", err)
	}

	return res, nil
}
