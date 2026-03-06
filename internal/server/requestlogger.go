package server

import (
	"strings"
	"time"

	obslogging "miconsul/internal/observability/logging"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/trace"
)

func RequestLoggerMiddleware(logger obslogging.Logger) func(c fiber.Ctx) error {
	return func(c fiber.Ctx) error {
		if !logger.Enabled() {
			return c.Next()
		}

		path := c.Path()
		if strings.HasPrefix(path, "/public/") || path == "/favicon.ico" || path == "/metrics" {
			return c.Next()
		}

		startedAt := time.Now()
		err := c.Next()

		routePath := path
		if route := c.Route(); route != nil && route.Path != "" {
			routePath = route.Path
		}

		statusCode := c.Response().StatusCode()
		method := c.Method()
		durationMS := float64(time.Since(startedAt).Microseconds()) / 1000.0

		traceID := ""
		spanCtx := trace.SpanFromContext(c.Context()).SpanContext()
		if spanCtx.IsValid() {
			traceID = spanCtx.TraceID().String()
		}

		logger.EmitHTTP(c.Context(), obslogging.HTTPEvent{
			Route:      routePath,
			Method:     method,
			Status:     statusCode,
			DurationMS: durationMS,
			TraceID:    traceID,
			Err:        err,
		})

		return err
	}
}
