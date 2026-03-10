package database

import (
	"net/url"
	"strings"
	"testing"
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
