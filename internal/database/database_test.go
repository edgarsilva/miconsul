package database

import (
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"miconsul/internal/lib/appenv"

	lognoop "go.opentelemetry.io/otel/log/noop"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func TestBuildSQLiteDSNAddsDefaultsForFilePath(t *testing.T) {
	dsn := buildSQLiteDSN("database/app.sqlite", nil)

	base, params := splitDSN(t, dsn)
	if base != "database/app.sqlite" {
		t.Fatalf("expected base path database/app.sqlite, got %q", base)
	}

	assertDefaultSQLiteParams(t, params)
}

func TestBuildSQLiteDSNPreservesExistingMode(t *testing.T) {
	dsn := buildSQLiteDSN("file:miconsul_test", SQLiteOptions{"mode": "memory", "cache": "shared"})

	base, params := splitDSN(t, dsn)
	if base != "file:miconsul_test" {
		t.Fatalf("expected base path file:miconsul_test, got %q", base)
	}

	if got := params.Get("mode"); got != "memory" {
		t.Fatalf("expected mode=memory, got %q", got)
	}
	if got := params.Get("cache"); got != "shared" {
		t.Fatalf("expected cache=shared, got %q", got)
	}

	assertDefaultSQLiteParamsExceptMode(t, params)
}

func TestBuildSQLiteDSNMergesExistingQueryWithoutDuplicates(t *testing.T) {
	dsn := buildSQLiteDSN("database/app.sqlite?cache=private&_busy_timeout=9000", nil)

	_, params := splitDSN(t, dsn)
	if got := params.Get("cache"); got != "private" {
		t.Fatalf("expected cache=private, got %q", got)
	}
	if got := params.Get("_busy_timeout"); got != "9000" {
		t.Fatalf("expected _busy_timeout=9000, got %q", got)
	}
	if got := len(params["_busy_timeout"]); got != 1 {
		t.Fatalf("expected no duplicate _busy_timeout key, got %d entries", got)
	}
	if got := len(params["mode"]); got != 1 {
		t.Fatalf("expected no duplicate mode key, got %d entries", got)
	}
}

func TestBuildSQLiteDSNAppliesOptionOverrides(t *testing.T) {
	dsn := buildSQLiteDSN("database/app.sqlite", SQLiteOptions{"_busy_timeout": "3000", "cache": "shared"})

	_, params := splitDSN(t, dsn)
	if got := params.Get("_busy_timeout"); got != "3000" {
		t.Fatalf("expected _busy_timeout=3000, got %q", got)
	}
	if got := params.Get("cache"); got != "shared" {
		t.Fatalf("expected cache=shared, got %q", got)
	}
}

func TestBuildSQLiteDSNUsesDefaultPathWhenBlank(t *testing.T) {
	dsn := buildSQLiteDSN("   ", nil)
	base, params := splitDSN(t, dsn)
	if base != "database/app.sqlite" {
		t.Fatalf("expected default base path, got %q", base)
	}
	assertDefaultSQLiteParams(t, params)
}

func TestBuildSQLiteDSNIgnoresBlankOptionKeysAndValues(t *testing.T) {
	dsn := buildSQLiteDSN("database/app.sqlite", SQLiteOptions{"": "x", "cache": "", "mode": "memory"})
	_, params := splitDSN(t, dsn)
	if got := params.Get("mode"); got != "memory" {
		t.Fatalf("expected mode override to memory, got %q", got)
	}
	if got := params.Get("cache"); got != "" {
		t.Fatalf("expected blank cache option to be ignored, got %q", got)
	}
}

func splitDSN(t *testing.T, dsn string) (string, url.Values) {
	t.Helper()

	base, rawQuery, found := strings.Cut(dsn, "?")
	if !found {
		t.Fatalf("expected query string in DSN: %q", dsn)
	}

	params, err := url.ParseQuery(rawQuery)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}

	return base, params
}

func assertDefaultSQLiteParams(t *testing.T, params url.Values) {
	t.Helper()

	for key, value := range defaultSQLiteParams {
		if got := params.Get(key); got != value {
			t.Fatalf("expected %s=%s, got %q", key, value, got)
		}
	}
}

func assertDefaultSQLiteParamsExceptMode(t *testing.T, params url.Values) {
	t.Helper()

	for key, value := range defaultSQLiteParams {
		if key == "mode" {
			continue
		}
		if got := params.Get(key); got != value {
			t.Fatalf("expected %s=%s, got %q", key, value, got)
		}
	}
}

func TestNewTracesQueriesWithInjectedProvider(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	db, err := New(&appenv.Env{DBPath: filepath.Join(t.TempDir(), "trace.sqlite")}, lognoop.NewLoggerProvider(), tp, nil)
	if err != nil {
		t.Fatalf("new database: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Exec("SELECT 1").Error; err != nil {
		t.Fatalf("exec query: %v", err)
	}

	if len(recorder.Ended()) == 0 {
		t.Fatal("expected a query span from the injected tracer provider")
	}
}

func TestNewRequiresTracerProvider(t *testing.T) {
	if _, err := New(&appenv.Env{DBPath: filepath.Join(t.TempDir(), "x.sqlite")}, lognoop.NewLoggerProvider(), nil, nil); err == nil {
		t.Fatal("expected nil tracer provider error")
	}
}

func TestNewGormLoggerAndDatabaseHelpers(t *testing.T) {
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
