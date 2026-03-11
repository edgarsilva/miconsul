package database

import (
	"bytes"
	"context"
	"log"
	"testing"
	"time"

	"miconsul/internal/lib/appenv"
	obslogging "miconsul/internal/observability/logging"

	"go.uber.org/zap"
	"gorm.io/gorm/logger"
)

func TestNewLoggerAndDatabaseHelpers(t *testing.T) {
	devLogger := NewLogger(&appenv.Env{Environment: appenv.EnvironmentDevelopment}, obslogging.Logger{})
	if devLogger == nil {
		t.Fatalf("expected development logger")
	}

	prodLogger := NewLogger(&appenv.Env{Environment: appenv.EnvironmentProduction}, obslogging.Logger{})
	if prodLogger == nil {
		t.Fatalf("expected production logger")
	}

	if err := ApplyMigrations(nil, nil); err == nil {
		t.Fatalf("expected migrations to fail for nil database")
	}
	if err := ApplyMigrationsSilent(&Database{}, nil); err == nil {
		t.Fatalf("expected silent migrations to fail for nil sql handle")
	}

	if _, err := New(nil, obslogging.Logger{}, nil); err == nil {
		t.Fatalf("expected New to fail with nil env")
	}

	d := &Database{}
	if d.GormDB() != nil {
		t.Fatalf("expected nil gorm db for empty database wrapper")
	}
}

func TestZapAndNoopLoggerAdapters(t *testing.T) {
	z, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("new zap logger: %v", err)
	}
	defer z.Sync()

	adapter := ZapLogger{l: z.Sugar()}
	adapter.Printf("hello %s", "db")

	NoopLogger{}.Printf("ignored")
	NoopLogger{}.Fatalf("ignored")
}

func TestGormObsLoggerDelegatesAndTrace(t *testing.T) {
	buf := &bytes.Buffer{}
	base := logger.New(log.New(buf, "", 0), logger.Config{LogLevel: logger.Info})
	wrapped := NewGormObsLogger(base, obslogging.Logger{})

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
