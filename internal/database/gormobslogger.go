package database

import (
	"context"
	"regexp"
	"strings"
	"time"

	obslogging "miconsul/internal/observability/logging"

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
	base logger.Interface
}

func NewGormObsLogger(base logger.Interface) logger.Interface {
	if base == nil {
		base = logger.Default
	}

	return GormObsLogger{base: base}
}

func (l GormObsLogger) LogMode(level logger.LogLevel) logger.Interface {
	return GormObsLogger{base: l.base.LogMode(level)}
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
	traceID := traceIDFromContext(ctx)
	durationMS := float64(time.Since(begin).Microseconds()) / 1000.0

	obslogging.EmitDBQuery(ctx, obslogging.DBQueryEvent{
		SQL:        cleanSQL,
		Rows:       rows,
		DurationMS: durationMS,
		TraceID:    traceID,
		Err:        err,
	})
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
