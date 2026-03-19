package appenv

import (
	"os"
	"testing"
	"time"

	"github.com/edgarsilva/simpleenv"
)

func TestEnvHelpers(t *testing.T) {
	t.Run("nil env returns false for all helpers", func(t *testing.T) {
		var env *Env
		if env.IsDevelopment() {
			t.Fatalf("expected IsDevelopment false for nil env")
		}
		if env.IsTest() {
			t.Fatalf("expected IsTest false for nil env")
		}
		if env.IsDevOrTest() {
			t.Fatalf("expected IsDevOrTest false for nil env")
		}
		if env.IsProduction() {
			t.Fatalf("expected IsProduction false for nil env")
		}
		if env.IsValidEnvironment() {
			t.Fatalf("expected IsValidEnvironment false for nil env")
		}
	})

	t.Run("environment helpers map correctly", func(t *testing.T) {
		env := &Env{Environment: EnvironmentDevelopment}
		if !env.IsDevelopment() || !env.IsDevOrTest() {
			t.Fatalf("expected development helpers true")
		}
		if env.IsProduction() || env.IsTest() {
			t.Fatalf("expected non-production/test helpers false")
		}

		env.Environment = EnvironmentTest
		if !env.IsTest() || !env.IsDevOrTest() {
			t.Fatalf("expected test helpers true")
		}

		env.Environment = EnvironmentProduction
		if !env.IsProduction() {
			t.Fatalf("expected IsProduction true")
		}
		if env.IsDevOrTest() {
			t.Fatalf("expected IsDevOrTest false in production")
		}

		env.Environment = "invalid"
		if env.IsValidEnvironment() {
			t.Fatalf("expected invalid environment to fail validation")
		}
	})
}

func TestEnvLoadOptionalDefaults(t *testing.T) {
	setRequiredEnv(t)

	env := &Env{
		AppShutdownTimeout: 10 * time.Second,
		RateLimiterEnabled: true,
		OTelServiceName:    "miconsul",
		OTelTracerServer:   "miconsul.server",
		OTelTracerAuth:     "miconsul.auth",
		JobsEnabled:        false,
		JobsUIEnabled:      false,
		ValkeyHost:         "127.0.0.1",
		ValkeyPort:         6379,
		ValkeyDB:           0,
	}

	if err := simpleenv.Load(env); err != nil {
		t.Fatalf("load env: %v", err)
	}

	if env.AppShutdownTimeout != 10*time.Second {
		t.Fatalf("expected default shutdown timeout 10s, got %v", env.AppShutdownTimeout)
	}
	if !env.RateLimiterEnabled {
		t.Fatalf("expected default rate limiter enabled true")
	}
	if env.OTelServiceName != "miconsul" {
		t.Fatalf("expected default OTel service name, got %q", env.OTelServiceName)
	}
	if env.JobsEnabled {
		t.Fatalf("expected default jobs enabled false")
	}
	if env.JobsUIEnabled {
		t.Fatalf("expected default jobs ui enabled false")
	}
	if env.ValkeyHost != "127.0.0.1" {
		t.Fatalf("expected default valkey host, got %q", env.ValkeyHost)
	}
	if env.ValkeyPort != 6379 {
		t.Fatalf("expected default valkey port 6379, got %d", env.ValkeyPort)
	}
}

func TestEnvLoadOptionalOverrides(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("APP_SHUTDOWN_TIMEOUT", "25s")
	t.Setenv("RATE_LIMITER_ENABLED", "false")
	t.Setenv("OTEL_SERVICE_NAME", "  test-service  ")
	t.Setenv("OTEL_TRACER_SERVER", "  tracer.server  ")
	t.Setenv("OTEL_TRACER_AUTH", "  tracer.auth  ")
	t.Setenv("JOBS_ENABLED", "true")
	t.Setenv("JOBS_UI_ENABLED", "true")
	t.Setenv("VALKEY_HOST", "  valkey  ")
	t.Setenv("VALKEY_PORT", "6380")
	t.Setenv("VALKEY_DB", "2")

	env := &Env{
		AppShutdownTimeout: 10 * time.Second,
		RateLimiterEnabled: true,
		OTelServiceName:    "miconsul",
		OTelTracerServer:   "miconsul.server",
		OTelTracerAuth:     "miconsul.auth",
		JobsEnabled:        false,
		JobsUIEnabled:      false,
		ValkeyHost:         "127.0.0.1",
		ValkeyPort:         6379,
		ValkeyDB:           0,
	}

	if err := simpleenv.Load(env); err != nil {
		t.Fatalf("load env: %v", err)
	}

	if env.AppShutdownTimeout != 25*time.Second {
		t.Fatalf("expected parsed shutdown timeout 25s, got %v", env.AppShutdownTimeout)
	}
	if env.RateLimiterEnabled {
		t.Fatalf("expected parsed rate limiter enabled false")
	}
	if env.OTelServiceName != "test-service" {
		t.Fatalf("expected trimmed OTel service name, got %q", env.OTelServiceName)
	}
	if env.OTelTracerServer != "tracer.server" {
		t.Fatalf("expected trimmed tracer server, got %q", env.OTelTracerServer)
	}
	if env.OTelTracerAuth != "tracer.auth" {
		t.Fatalf("expected trimmed tracer auth, got %q", env.OTelTracerAuth)
	}
	if !env.JobsEnabled {
		t.Fatalf("expected parsed jobs enabled true")
	}
	if !env.JobsUIEnabled {
		t.Fatalf("expected parsed jobs ui enabled true")
	}
	if env.ValkeyHost != "valkey" {
		t.Fatalf("expected trimmed valkey host, got %q", env.ValkeyHost)
	}
	if env.ValkeyPort != 6380 {
		t.Fatalf("expected parsed valkey port 6380, got %d", env.ValkeyPort)
	}
	if env.ValkeyDB != 2 {
		t.Fatalf("expected parsed valkey db 2, got %d", env.ValkeyDB)
	}
}

func TestEnvLoadInvalidDuration(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("APP_SHUTDOWN_TIMEOUT", "not-a-duration")

	env := &Env{AppShutdownTimeout: 10 * time.Second}
	if err := simpleenv.Load(env); err == nil {
		t.Fatalf("expected invalid duration error")
	}
}

func TestNewLoadsEnvironment(t *testing.T) {
	setRequiredEnv(t)

	env, err := New()
	if err != nil {
		t.Fatalf("unexpected New error: %v", err)
	}
	if env == nil {
		t.Fatalf("expected non-nil env")
	}
	if env.AppName != "miconsul" {
		t.Fatalf("expected app name from env, got %q", env.AppName)
	}
	if env.RateLimiterEnabled != true {
		t.Fatalf("expected default rate limiter enabled true")
	}
}

func TestNewReturnsErrorOnLoadFailure(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("COOKIE_SECRET", "too-short")

	env, err := New()
	if err == nil {
		t.Fatalf("expected New to return error when env vars are invalid")
	}
	if env != nil {
		t.Fatalf("expected nil env on load failure")
	}
}

func TestNewReturnsErrorWhenRequiredEnvMissing(t *testing.T) {
	setRequiredEnv(t)
	if err := os.Unsetenv("APP_NAME"); err != nil {
		t.Fatalf("unset APP_NAME: %v", err)
	}

	env, err := New()
	if err == nil {
		t.Fatalf("expected New to return error when required env vars are missing")
	}
	if env != nil {
		t.Fatalf("expected nil env on load failure")
	}
}

func setRequiredEnv(t *testing.T) {
	t.Helper()
	clearEnv(t,
		"APP_SHUTDOWN_TIMEOUT",
		"RATE_LIMITER_ENABLED",
		"CACHE_DB_PATH",
		"LOGTO_RESOURCE",
		"LOGTO_URL",
		"LOGTO_APP_ID",
		"LOGTO_APP_SECRET",
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_INSECURE",
		"OTEL_SERVICE_NAME",
		"OTEL_TRACER_SERVER",
		"OTEL_TRACER_AUTH",
		"JOBS_ENABLED",
		"JOBS_UI_ENABLED",
		"VALKEY_HOST",
		"VALKEY_PORT",
		"VALKEY_PASSWORD",
		"VALKEY_DB",
	)

	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_NAME", "miconsul")
	t.Setenv("APP_PROTOCOL", "http")
	t.Setenv("APP_DOMAIN", "localhost")
	t.Setenv("APP_VERSION", "test")
	t.Setenv("APP_PORT", "3000")
	t.Setenv("COOKIE_SECRET", "12345678901234567890123456789012")
	t.Setenv("JWT_SECRET", "test-jwt-secret")
	t.Setenv("DB_PATH", "tmp/db.sqlite")
	t.Setenv("SESSION_DB_PATH", "tmp/session.sqlite")
	t.Setenv("EMAIL_SENDER", "sender")
	t.Setenv("EMAIL_SECRET", "secret")
	t.Setenv("EMAIL_FROM_ADDRESS", "sender@example.com")
	t.Setenv("EMAIL_SMTP_URL", "smtp://localhost:1025")
	t.Setenv("GOOSE_DRIVER", "sqlite")
	t.Setenv("GOOSE_DBSTRING", "tmp/db.sqlite")
	t.Setenv("GOOSE_MIGRATION_DIR", "internal/db/migrations")
	t.Setenv("ASSETS_DIR", "static")
}

func clearEnv(t *testing.T, keys ...string) {
	t.Helper()

	for _, key := range keys {
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	}
}
