package database

import (
	"context"
	"regexp"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/trace"
	gormlogger "gorm.io/gorm/logger"
)

const maxSQLLogLength = 512

var (
	singleQuotedLiteralRE = regexp.MustCompile(`'([^'\\]|\\.)*'`)
	doubleQuotedLiteralRE = regexp.MustCompile(`"([^"\\]|\\.)*"`)
	numberLiteralRE       = regexp.MustCompile(`\b\d+(\.\d+)?\b`)
	boolLiteralRE         = regexp.MustCompile(`\b(true|false)\b`)
	whitespaceRE          = regexp.MustCompile(`\s+`)
)

type GormOtelLogger struct {
	gormLogger gormlogger.Interface
	otelLogger otellog.Logger
}

func NewGormOtelLogger(gormLogger gormlogger.Interface, otelLogger otellog.Logger) gormlogger.Interface {
	if gormLogger == nil {
		gormLogger = gormlogger.Default
	}

	return GormOtelLogger{gormLogger: gormLogger, otelLogger: otelLogger}
}

func (l GormOtelLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return GormOtelLogger{gormLogger: l.gormLogger.LogMode(level), otelLogger: l.otelLogger}
}

func (l GormOtelLogger) Info(ctx context.Context, msg string, data ...any) {
	l.gormLogger.Info(ctx, msg, data...)
}

func (l GormOtelLogger) Warn(ctx context.Context, msg string, data ...any) {
	l.gormLogger.Warn(ctx, msg, data...)
}

func (l GormOtelLogger) Error(ctx context.Context, msg string, data ...any) {
	l.gormLogger.Error(ctx, msg, data...)
}

func (l GormOtelLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, rows := fc()
	fcCached := func() (string, int64) {
		return sql, rows
	}
	l.gormLogger.Trace(ctx, begin, fcCached, err)

	cleanSQL := sanitizeSQLForLog(sql)
	dbOperation := extractDBOperation(sql)
	traceID := traceIDFromContext(ctx)
	durationMS := float64(time.Since(begin).Microseconds()) / 1000.0

	rec := otellog.Record{}
	rec.SetTimestamp(time.Now())
	rec.SetObservedTimestamp(time.Now())
	rec.SetEventName("db_query")
	rec.SetBody(attribute.StringValue("db_query"))

	if err != nil {
		rec.SetSeverity(otellog.SeverityError)
		rec.SetSeverityText("ERROR")
	} else {
		rec.SetSeverity(otellog.SeverityInfo)
		rec.SetSeverityText("INFO")
	}

	attrs := []attribute.KeyValue{
		attribute.String("event", "db_query"),
		attribute.String("db_operation", dbOperation),
		attribute.String("sql", cleanSQL),
		attribute.Int64("rows", rows),
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

	l.otelLogger.Emit(ctx, rec)
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
