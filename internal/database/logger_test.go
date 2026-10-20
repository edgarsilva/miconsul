package database

import (
	"bytes"
	"context"
	"log"
	"testing"
	"time"

	"miconsul/internal/lib/appenv"

	lognoop "go.opentelemetry.io/otel/log/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"gorm.io/gorm/logger"
)

func TestNewLoggerAndDatabaseHelpers(t *testing.T) {
	devLogger := NewGormLogger(&appenv.Env{Environment: appenv.EnvironmentDevelopment}, lognoop.Logger{})
	if devLogger == nil {
		t.Fatalf("expected development logger")
	}

	prodLogger := NewGormLogger(&appenv.Env{Environment: appenv.EnvironmentProduction}, lognoop.Logger{})
	if prodLogger == nil {
		t.Fatalf("expected production logger")
	}

	if err := ApplyMigrations(nil); err == nil {
		t.Fatalf("expected migrations to fail for nil database")
	}
	if err := ApplyMigrationsSilent(&Database{}); err == nil {
		t.Fatalf("expected silent migrations to fail for nil sql handle")
	}

	if _, err := New(nil, lognoop.NewLoggerProvider(), tracenoop.NewTracerProvider(), nil); err == nil {
		t.Fatalf("expected New to fail with nil env")
	}

	d := &Database{}
	if d.GormDB() != nil {
		t.Fatalf("expected nil gorm db for empty database wrapper")
	}
}

func TestGooseAndNoopLoggerAdapters(t *testing.T) {
	adapter := NewGooseLogger(lognoop.Logger{})
	adapter.Printf("hello %s", "db")
	adapter.Fatalf("fatal %s", "db")

	NoopLogger{}.Printf("ignored")
	NoopLogger{}.Fatalf("ignored")
}

func TestGormOtelLoggerDelegatesAndTrace(t *testing.T) {
	buf := &bytes.Buffer{}
	base := logger.New(log.New(buf, "", 0), logger.Config{LogLevel: logger.Info})
	wrapped := NewGormOtelLogger(base, lognoop.Logger{})

	wrapped.Info(context.Background(), "info %s", "msg")
	wrapped.Warn(context.Background(), "warn")
	wrapped.Error(context.Background(), "err")
	wrapped.Trace(context.Background(), time.Now().Add(-5*time.Millisecond), func() (string, int64) {
		return "SELECT * FROM users WHERE id = 1", 1
	}, nil)

	if buf.Len() == 0 {
		t.Fatalf("expected base logger output after delegated calls")
	}
}
