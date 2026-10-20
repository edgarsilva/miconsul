package database

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
	"time"

	lognoop "go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/trace"
	gormlogger "gorm.io/gorm/logger"
)

func TestSanitizeSQLForLog(t *testing.T) {
	query := "SELECT * FROM users WHERE email='a@example.com' AND id=42 AND active=true"
	got := sanitizeSQLForLog(query)
	if strings.Contains(got, "a@example.com") || strings.Contains(got, "42") || strings.Contains(got, "true") {
		t.Fatalf("expected literals to be scrubbed, got %q", got)
	}
	if !strings.Contains(got, "SELECT * FROM users") {
		t.Fatalf("expected SQL shape retained, got %q", got)
	}
}

func TestSanitizeSQLForLogTruncates(t *testing.T) {
	query := "SELECT " + strings.Repeat("a", maxSQLLogLength+50)
	got := sanitizeSQLForLog(query)
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("expected truncated query to end with ellipsis, got %q", got)
	}
}

func TestExtractDBOperation(t *testing.T) {
	cases := map[string]string{
		"":                         "UNKNOWN",
		"   ":                      "UNKNOWN",
		"SELECT 1":                 "SELECT",
		"insert into users values": "INSERT",
		"update users set":         "UPDATE",
		"delete from users":        "DELETE",
		"pragma table_info(users)": "OTHER",
	}

	for input, expected := range cases {
		if got := extractDBOperation(input); got != expected {
			t.Fatalf("extractDBOperation(%q): expected %q, got %q", input, expected, got)
		}
	}
}

func TestTraceIDFromContext(t *testing.T) {
	if got := traceIDFromContext(nil); got != "" {
		t.Fatalf("expected empty trace id for nil context, got %q", got)
	}

	traceID, err := trace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("parse trace id: %v", err)
	}
	spanID, err := trace.SpanIDFromHex("0123456789abcdef")
	if err != nil {
		t.Fatalf("parse span id: %v", err)
	}
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled, Remote: true})
	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)

	if got := traceIDFromContext(ctx); got != traceID.String() {
		t.Fatalf("expected trace id %q, got %q", traceID.String(), got)
	}
}

func TestNewGormOtelLoggerNilGormLogger(t *testing.T) {
	l := NewGormOtelLogger(nil, lognoop.Logger{})
	if l == nil {
		t.Fatalf("expected non-nil gorm logger wrapper")
	}

	l2 := l.LogMode(gormlogger.Info)
	if l2 == nil {
		t.Fatalf("expected log mode logger to be non-nil")
	}
}

func TestGormOtelLoggerDelegatesAndTrace(t *testing.T) {
	buf := &bytes.Buffer{}
	gormLogger := gormlogger.New(log.New(buf, "", 0), gormlogger.Config{LogLevel: gormlogger.Info})
	wrapped := NewGormOtelLogger(gormLogger, lognoop.Logger{})

	wrapped.Info(context.Background(), "info %s", "msg")
	wrapped.Warn(context.Background(), "warn")
	wrapped.Error(context.Background(), "err")
	wrapped.Trace(context.Background(), time.Now().Add(-5*time.Millisecond), func() (string, int64) {
		return "SELECT * FROM users WHERE id = 1", 1
	}, nil)

	if buf.Len() == 0 {
		t.Fatalf("expected gorm logger output after delegated calls")
	}
}
