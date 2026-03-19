package server

import (
	"context"
	"testing"

	"miconsul/internal/database"
	"miconsul/internal/jobs"
	"miconsul/internal/lib/appenv"
	"miconsul/internal/lib/localize"
	obslogging "miconsul/internal/observability/logging"
	obsmetrics "miconsul/internal/observability/metrics"

	"go.opentelemetry.io/otel"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWithOptionSetters(t *testing.T) {
	s := &Server{}
	env := &appenv.Env{Environment: appenv.EnvironmentDevelopment, CookieSecret: "12345678901234567890123456789012"}
	if err := WithEnv(env)(s); err != nil {
		t.Fatalf("with env error: %v", err)
	}

	db, err := gorm.Open(sqlite.Open("file:server_opts?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := WithDatabase(&database.Database{DB: db})(s); err != nil {
		t.Fatalf("with db error: %v", err)
	}

	if err := WithTracer(otel.Tracer("test"))(s); err != nil {
		t.Fatalf("with tracer error: %v", err)
	}

	cache := &cacheStub{}
	if err := WithCache(cache)(s); err != nil {
		t.Fatalf("with cache error: %v", err)
	}
	if s.Cache == nil {
		t.Fatalf("expected cache to be set")
	}

	jobsRuntime := &jobs.Runtime{}
	if err := WithJobs(jobsRuntime)(s); err != nil {
		t.Fatalf("with jobs runtime error: %v", err)
	}
	if got := s.JobsRuntime(); got != jobsRuntime {
		t.Fatalf("expected jobs runtime to be set")
	}

	if err := WithEnv(nil)(s); err == nil {
		t.Fatalf("expected WithEnv nil error")
	}
	if err := WithDatabase(nil)(s); err == nil {
		t.Fatalf("expected WithDatabase nil error")
	}
	if err := WithTracer(nil)(s); err != nil {
		t.Fatalf("expected WithTracer nil noop: %v", err)
	}
	if err := WithLocalizer(nil)(s); err != nil {
		t.Fatalf("expected WithLocalizer nil noop: %v", err)
	}
	if err := WithWorkPool(nil)(s); err != nil {
		t.Fatalf("expected WithWorkPool nil noop: %v", err)
	}
	if err := WithCache(nil)(s); err != nil {
		t.Fatalf("expected WithCache nil noop: %v", err)
	}
	if err := WithJobs(nil)(s); err != nil {
		t.Fatalf("expected WithJobs nil noop: %v", err)
	}

	if err := WithLocalizer(localize.New("es-MX", "en-US"))(s); err != nil {
		t.Fatalf("set localizer: %v", err)
	}
	if err := WithMetrics(obsmetrics.HTTPMetrics{})(s); err != nil {
		t.Fatalf("set metrics: %v", err)
	}
	if err := WithRequestLogger(obslogging.Logger{})(s); err != nil {
		t.Fatalf("set request logger: %v", err)
	}
}

func TestServerHelperPassthroughs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:server_helper_passthrough?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	env := &appenv.Env{Environment: appenv.EnvironmentDevelopment, CookieSecret: "12345678901234567890123456789012"}
	s := &Server{Env: env, DB: &database.Database{DB: db}, Tracer: otel.Tracer("test")}

	ctx, span := s.Trace(context.Background(), "unit-span")
	if ctx == nil || span == nil {
		t.Fatalf("expected trace to return context and span")
	}
	span.End()

	if got := s.AppEnv(); got != env {
		t.Fatalf("expected AppEnv passthrough")
	}
	if got := s.GormDB(); got == nil {
		t.Fatalf("expected gorm db passthrough")
	}
}
