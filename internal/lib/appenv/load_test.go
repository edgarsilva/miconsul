package appenv

import (
	"os"
	"testing"
	"time"

	"github.com/edgarsilva/simpleenv"
)

func TestEnvLoadOptionalDefaults(t *testing.T) {
	setRequiredEnv(t)

	env := &Env{
		AppShutdownTimeout: 10 * time.Second,
		RateLimiterEnabled: true,
		OTelServiceName:    "miconsul",
		OTelTracerServer:   "miconsul.server",
		OTelTracerAuth:     "miconsul.auth",
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
}

func TestEnvLoadOptionalOverrides(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("APP_SHUTDOWN_TIMEOUT", "25s")
	t.Setenv("RATE_LIMITER_ENABLED", "false")
	t.Setenv("OTEL_SERVICE_NAME", "  test-service  ")
	t.Setenv("OTEL_TRACER_SERVER", "  tracer.server  ")
	t.Setenv("OTEL_TRACER_AUTH", "  tracer.auth  ")

	env := &Env{
		AppShutdownTimeout: 10 * time.Second,
		RateLimiterEnabled: true,
		OTelServiceName:    "miconsul",
		OTelTracerServer:   "miconsul.server",
		OTelTracerAuth:     "miconsul.auth",
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
