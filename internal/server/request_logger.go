package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/trace"
)

func RequestLoggerMiddleware(logger otellog.Logger) func(c fiber.Ctx) error {
	return func(c fiber.Ctx) error {
		if logger == nil || !logger.Enabled(c.Context(), otellog.EnabledParameters{}) {
			return c.Next()
		}

		path := c.Path()
		if strings.HasPrefix(path, "/public/") || path == "/favicon.ico" || path == "/readyz" {
			return c.Next()
		}

		startedAt := time.Now()

		// A handler panic skips the code below, so log it here as a 500 and
		// re-panic for the recover middleware to build the response.
		defer func() {
			if r := recover(); r != nil {
				emitRequestLog(c, logger, path, startedAt, fiber.StatusInternalServerError, fmt.Errorf("panic: %v", r))
				panic(r)
			}
		}()

		err := c.Next()
		emitRequestLog(c, logger, path, startedAt, c.Response().StatusCode(), err)

		return err
	}
}

// emitRequestLog emits the http_request record for a finished request.
func emitRequestLog(c fiber.Ctx, logger otellog.Logger, path string, startedAt time.Time, statusCode int, err error) {
	routePath := path
	if route := c.Route(); route != nil && route.Path != "" {
		routePath = route.Path
	}

	method := c.Method()
	durationMS := float64(time.Since(startedAt).Microseconds()) / 1000.0

	traceID := ""
	spanCtx := trace.SpanFromContext(c.Context()).SpanContext()
	if spanCtx.IsValid() {
		traceID = spanCtx.TraceID().String()
	}

	rec := otellog.Record{}
	rec.SetTimestamp(time.Now())
	rec.SetObservedTimestamp(time.Now())
	rec.SetEventName("http_request")
	rec.SetBody(attribute.StringValue("http_request"))

	if statusCode >= 500 {
		rec.SetSeverity(otellog.SeverityError)
		rec.SetSeverityText("ERROR")
	} else {
		rec.SetSeverity(otellog.SeverityInfo)
		rec.SetSeverityText("INFO")
	}

	attrs := []attribute.KeyValue{
		attribute.String("event", "http_request"),
		attribute.String("route", routePath),
		attribute.String("method", method),
		attribute.Int("status", statusCode),
		attribute.Float64("duration_ms", durationMS),
	}
	if traceID != "" {
		attrs = append(attrs, attribute.String("trace_id", traceID))
	}
	if err != nil {
		rec.SetErr(err)
		attrs = append(attrs, attribute.String("error", err.Error()))
	}
	rec.AddAttributes(attrs...)

	logger.Emit(c.Context(), rec)
}
