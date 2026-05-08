package server

import (
	"context"
	"fmt"
	"time"

	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/trace"
)

// Trace starts a span with the configured tracer and returns updated context.
func (s *Server) Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if ctx == nil {
		ctx = context.Background()
	}

	if s.Tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}

	ctx, span := s.Tracer.Start(ctx, spanName, opts...)
	return ctx, span
}

// SendToWorker passes fn as a job for a worker in the workpool, to be executed as a go routine
// when the a worker is available. It propagates trace context into the worker and recovers panics.
func (s *Server) SendToWorker(ctx context.Context, fn func()) error {
	workerCtx, span := s.Trace(ctx, "server/SendToWorker")
	defer span.End()

	wrapped := func() {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("panic in worker: %v", r)
				span.RecordError(err)
				s.emitWorkerError(workerCtx, err)
			}
		}()
		fn()
	}

	if s.wp == nil {
		wrapped()
		return nil
	}

	if err := s.wp.Submit(wrapped); err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}

func (s *Server) emitWorkerError(ctx context.Context, err error) {
	if err == nil || !s.RequestLog.Enabled() {
		return
	}

	rec := otellog.Record{}
	rec.SetTimestamp(time.Now())
	rec.SetObservedTimestamp(time.Now())
	rec.SetEventName("worker_panic")
	rec.SetBody(otellog.StringValue("worker_panic"))
	rec.SetSeverity(otellog.SeverityError)
	rec.SetSeverityText("ERROR")
	rec.SetErr(err)
	rec.AddAttributes(
		otellog.String("event", "worker_panic"),
		otellog.String("error", err.Error()),
	)

	s.RequestLog.Emit(ctx, rec)
}

func emitStartupBootstrapLog(s *Server) {
	if !s.RequestLog.Enabled() {
		return
	}

	rec := otellog.Record{}
	rec.SetTimestamp(time.Now())
	rec.SetObservedTimestamp(time.Now())
	rec.SetEventName("server_startup")
	rec.SetBody(otellog.StringValue("server_startup"))
	rec.SetSeverity(otellog.SeverityInfo)
	rec.SetSeverityText("INFO")
	rec.AddAttributes(
		otellog.String("event", "server_startup"),
		otellog.String("started_at", s.StartedAt.UTC().Format(time.RFC3339)),
		otellog.String("ready_at", s.ReadyAt.UTC().Format(time.RFC3339)),
		otellog.Int64("bootstrap_duration_ms", s.BootstrapDuration.Milliseconds()),
		otellog.String("version", s.Env.AppVersion),
		otellog.String("environment", string(s.Env.Environment)),
	)

	s.RequestLog.Emit(context.Background(), rec)
}
