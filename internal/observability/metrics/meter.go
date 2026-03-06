package metrics

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"miconsul/internal/lib/appenv"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	metricapi "go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

type Meter struct {
	PromHTTPDuration *prometheus.HistogramVec
	PromHTTPRequests *prometheus.CounterVec

	HTTPDuration metricapi.Float64Histogram
	HTTPRequests metricapi.Int64Counter
}

var registerPromMetricsOnce sync.Once

func New(ctx context.Context, env *appenv.Env) (Meter, func() error, error) {
	provider, shutdownFn, err := NewMeterProvider(ctx, env)
	if err != nil {
		return Meter{}, shutdownFn, err
	}

	otelHTTPDuration, err := provider.Float64Histogram(
		"http_request_duration_seconds",
		metricapi.WithDescription("HTTP request duration in seconds."),
	)
	if err != nil {
		return Meter{}, shutdownFn, err
	}

	otelHTTPRequests, err := provider.Int64Counter(
		"http_requests_total",
		metricapi.WithDescription("Total number of HTTP requests."),
	)
	if err != nil {
		return Meter{}, shutdownFn, err
	}

	promHTTPDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "HTTP request duration in seconds.",
			Buckets: []float64{
				0.005, 0.01, 0.025, 0.05, 0.075,
				0.1, 0.15, 0.2, 0.3, 0.5,
				0.75, 1, 1.5, 2, 3, 5, 10,
			},
		},
		[]string{"route", "method", "status_group"},
	)

	promHTTPRequests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"route", "method", "status_group"},
	)

	registerPromMetricsOnce.Do(func() {
		prometheus.MustRegister(promHTTPDuration, promHTTPRequests)
	})

	return Meter{
		PromHTTPDuration: promHTTPDuration,
		PromHTTPRequests: promHTTPRequests,
		HTTPDuration:     otelHTTPDuration,
		HTTPRequests:     otelHTTPRequests,
	}, shutdownFn, nil
}

func NewMeterProvider(ctx context.Context, env *appenv.Env) (metric.Meter, func() error, error) {
	if env == nil {
		return nil, nil, fmt.Errorf("otel metrics: environment config is nil")
	}

	if env.OTelOTLPEndpoint == "" {
		meter := otel.GetMeterProvider().Meter(env.AppName + ".metrics")
		return meter, func() error { return nil }, nil
	}

	endpoint := env.OTelOTLPEndpoint
	serviceName := env.OTelServiceName
	if serviceName == "" {
		serviceName = env.AppName
	}

	otlpOpts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(endpoint)}
	if env.OTelOTLPInsecure || isInternalOTLPEndpoint(endpoint) {
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
	meter := mp.Meter(env.AppName + ".metrics")

	shutdown := func() error {
		if err := mp.Shutdown(context.Background()); err != nil {
			return fmt.Errorf("otel metrics: shutdown meter provider: %w", err)
		}
		return nil
	}

	return meter, shutdown, nil
}

func isInternalOTLPEndpoint(endpoint string) bool {
	host := strings.ToLower(endpoint)
	return strings.Contains(host, "localhost") || strings.Contains(host, "127.0.0.1") || strings.Contains(host, "lgtm") || strings.Contains(host, "tempo")
}
