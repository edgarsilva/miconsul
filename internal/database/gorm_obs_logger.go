package database

import (
	"context"
	"regexp"
	"strings"
	"time"

	obslogging "miconsul/internal/observability/logging"

	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm/logger"
)

const maxSQLLogLength = 512

var (
	singleQuotedLiteralRE = regexp.MustCompile(`'([^'\\]|\\.)*'`)
	doubleQuotedLiteralRE = regexp.MustCompile(`"([^"\\]|\\.)*"`)
	numberLiteralRE       = regexp.MustCompile(`\b\d+(\.\d+)?\b`)
	boolLiteralRE         = regexp.MustCompile(`\b(true|false)\b`)
	whitespaceRE          = regexp.MustCompile(`\s+`)
)

type GormObsLogger struct {
	base      logger.Interface
	obsLogger obslogging.Logger
}

func NewGormObsLogger(base logger.Interface, obsLogger obslogging.Logger) logger.Interface {
	if base == nil {
		base = logger.Default
	}

	return GormObsLogger{base: base, obsLogger: obsLogger}
}

func (l GormObsLogger) LogMode(level logger.LogLevel) logger.Interface {
	return GormObsLogger{base: l.base.LogMode(level), obsLogger: l.obsLogger}
}

func (l GormObsLogger) Info(ctx context.Context, msg string, data ...any) {
	l.base.Info(ctx, msg, data...)
}

func (l GormObsLogger) Warn(ctx context.Context, msg string, data ...any) {
	l.base.Warn(ctx, msg, data...)
}

func (l GormObsLogger) Error(ctx context.Context, msg string, data ...any) {
	l.base.Error(ctx, msg, data...)
}

func (l GormObsLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, rows := fc()
	fcCached := func() (string, int64) {
		return sql, rows
	}
	l.base.Trace(ctx, begin, fcCached, err)

	cleanSQL := sanitizeSQLForLog(sql)
	dbOperation := extractDBOperation(sql)
	traceID := traceIDFromContext(ctx)
	durationMS := float64(time.Since(begin).Microseconds()) / 1000.0

	rec := otellog.Record{}
	rec.SetTimestamp(time.Now())
	rec.SetObservedTimestamp(time.Now())
	rec.SetEventName("db_query")
	rec.SetBody(otellog.StringValue("db_query"))

	if err != nil {
		rec.SetSeverity(otellog.SeverityError)
		rec.SetSeverityText("ERROR")
	} else {
		rec.SetSeverity(otellog.SeverityInfo)
		rec.SetSeverityText("INFO")
	}

	attrs := []otellog.KeyValue{
		otellog.String("event", "db_query"),
		otellog.String("db_operation", dbOperation),
		otellog.String("sql", cleanSQL),
		otellog.Int64("rows", rows),
		otellog.Float64("duration_ms", durationMS),
	}
	if traceID != "" {
		attrs = append(attrs, otellog.String("trace_id", traceID))
	}
	if err != nil {
		rec.SetErr(err)
		attrs = append(attrs, otellog.String("error", err.Error()))
	}
	rec.AddAttributes(attrs...)

	l.obsLogger.Emit(ctx, rec)
}

func traceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	spanCtx := trace.SpanFromContext(ctx).SpanContext()
	if !spanCtx.IsValid() {
		return ""
	}

	return spanCtx.TraceID().String()
}

func sanitizeSQLForLog(query string) string {
	if query == "" {
		return ""
	}

	query = singleQuotedLiteralRE.ReplaceAllString(query, "?")
	query = doubleQuotedLiteralRE.ReplaceAllString(query, "?")
	query = boolLiteralRE.ReplaceAllString(query, "?")
	query = numberLiteralRE.ReplaceAllString(query, "?")
	query = whitespaceRE.ReplaceAllString(strings.TrimSpace(query), " ")

	if len(query) > maxSQLLogLength {
		return query[:maxSQLLogLength] + "..."
	}

	return query
}

func extractDBOperation(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return "UNKNOWN"
	}

	parts := strings.Fields(query)
	if len(parts) == 0 {
		return "UNKNOWN"
	}

	op := strings.ToUpper(parts[0])
	switch op {
	case "SELECT", "INSERT", "UPDATE", "DELETE":
		return op
	default:
		return "OTHER"
	}
}
