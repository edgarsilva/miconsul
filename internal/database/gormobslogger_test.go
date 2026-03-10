package database

import (
	"context"
	"strings"
	"testing"

	obslogging "miconsul/internal/observability/logging"

	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm/logger"
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

func TestNewGormObsLoggerNilBase(t *testing.T) {
	l := NewGormObsLogger(nil, obslogging.Logger{})
	if l == nil {
		t.Fatalf("expected non-nil gorm logger wrapper")
	}

	l2 := l.LogMode(logger.Info)
	if l2 == nil {
		t.Fatalf("expected log mode logger to be non-nil")
	}
}

func TestDatabaseNilReceiverMethods(t *testing.T) {
	var d *Database
	sqlDB, err := d.SQLDB()
	if err != nil {
		t.Fatalf("expected nil receiver SQLDB to return nil,nil, got err=%v", err)
	}
	if sqlDB != nil {
		t.Fatalf("expected nil sql db handle for nil receiver")
	}

	if err := d.Close(); err != nil {
		t.Fatalf("expected nil receiver Close to be no-op, got %v", err)
	}
}
