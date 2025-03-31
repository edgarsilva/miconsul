package otel

import (
	"context"
	"fmt"
	"log"
	"miconsul/internal/lib/appenv"

	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	//"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
)

const (
	dsn = "https://bERP4WQyw5wRLuwfBcgVtg@api.uptrace.dev?grpc=4317"
)

func NewStdoutTracerProvider(ctx context.Context) *sdktrace.TracerProvider {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Panic(err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String("miconsul-fiberapp"),
			)),
	)

	otel.SetTracerProvider(tp)

	textPropagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)

	otel.SetTextMapPropagator(textPropagator)

	return tp
}

func NewUptraceTracerProvider(ctx context.Context, env *appenv.Env) (tp *sdktrace.TracerProvider, shutdownFn func()) {
	var (
		dsn         = env.UptraceEndpoint
		endpoint    = env.UptraceEndpoint
		appName     = env.AppName
		appEnv      = env.AppEnv
		appVersion  = env.AppVersion
		serviceName = appName + "_" + appEnv
	)

	if dsn == "" || endpoint == "" {
		log.Println("Failed to create UPTRACE exporter DSN or ENDPOINT env vars missing")
	}

	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithHeaders(map[string]string{
			"uptrace-dsn": dsn,
		}),
	)
	if err != nil {
		log.Panic(err)
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
		log.Panic(err)
	}

	bsp := sdktrace.NewBatchSpanProcessor(exporter,
		sdktrace.WithMaxQueueSize(10_000),
		sdktrace.WithMaxExportBatchSize(10_000),
	)

	tp = sdktrace.NewTracerProvider(
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

	return tp, func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			fmt.Println("Error shutting down tracer provider: ", err)
		}
	}
}
