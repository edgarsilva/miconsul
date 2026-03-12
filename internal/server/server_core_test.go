package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"
	"miconsul/internal/lib/cronjob"
	"miconsul/internal/lib/localize"
	obslogging "miconsul/internal/observability/logging"
	obsmetrics "miconsul/internal/observability/metrics"

	"github.com/panjf2000/ants/v2"
	"go.opentelemetry.io/otel"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestValidateCriticalDepsAndRuntimeConfig(t *testing.T) {
	if err := validateCriticalDeps(nil); err == nil {
		t.Fatalf("expected nil server validation error")
	}

	s := &Server{}
	if err := validateCriticalDeps(s); err == nil {
		t.Fatalf("expected missing env validation error")
	}

	s.Env = &appenv.Env{}
	if err := validateCriticalDeps(s); err == nil {
		t.Fatalf("expected missing db validation error")
	}

	db, err := gorm.Open(sqlite.Open("file:server_core_validate?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	s.DB = &database.Database{DB: db}
	if err := validateCriticalDeps(s); err == nil {
		t.Fatalf("expected missing tracer validation error")
	}

	s.Tracer = otel.Tracer("test")
	if err := validateCriticalDeps(s); err != nil {
		t.Fatalf("expected critical deps to pass: %v", err)
	}

	if err := validateRuntimeConfig(nil); err == nil {
		t.Fatalf("expected nil runtime validation error")
	}

	s.Env = &appenv.Env{Environment: appenv.Environment("invalid"), CookieSecret: "short"}
	if err := validateRuntimeConfig(s); err == nil {
		t.Fatalf("expected invalid APP_ENV error")
	}

	s.Env = &appenv.Env{Environment: appenv.EnvironmentDevelopment, CookieSecret: ""}
	if err := validateRuntimeConfig(s); err == nil {
		t.Fatalf("expected missing cookie secret error")
	}

	s.Env = &appenv.Env{Environment: appenv.EnvironmentDevelopment, CookieSecret: "1234567890123456789012345678901"}
	if err := validateRuntimeConfig(s); err == nil {
		t.Fatalf("expected short cookie secret error")
	}

	s.Env = &appenv.Env{Environment: appenv.EnvironmentDevelopment, CookieSecret: "12345678901234567890123456789012"}
	if err := validateRuntimeConfig(s); err != nil {
		t.Fatalf("expected runtime config to pass: %v", err)
	}
}

func TestSetupFunctionsAndOptions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:server_core_setup?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	env := &appenv.Env{
		Environment:        appenv.EnvironmentDevelopment,
		CookieSecret:       "12345678901234567890123456789012",
		SessionDBPath:      t.TempDir(),
		AppProtocol:        "https",
		RateLimiterEnabled: true,
		AppPort:            8080,
	}

	s := &Server{Env: env, DB: &database.Database{DB: db}, Tracer: otel.Tracer("test")}
	setupSessionStore(s)
	if s.SessionStore == nil {
		t.Fatalf("expected session store initialization")
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "public", ".well-known"), 0o755); err != nil {
		t.Fatalf("mkdir public dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "public", "favicon.ico"), []byte("ico"), 0o644); err != nil {
		t.Fatalf("write favicon: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir tmp: %v", err)
	}

	setupFiberApp(s)
	if s.App == nil {
		t.Fatalf("expected fiber app initialization")
	}

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
	if err := WithCronJob(nil)(s); err != nil {
		t.Fatalf("expected WithCronJob nil noop: %v", err)
	}
	if err := WithCache(nil)(s); err != nil {
		t.Fatalf("expected WithCache nil noop: %v", err)
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

func TestCronAndWorkerHelpers(t *testing.T) {
	s := &Server{cronJobKeys: map[string]struct{}{}}

	if err := s.AddCronJob("* * * * *", func() {}); err == nil {
		t.Fatalf("expected AddCronJob to fail without scheduler")
	}
	if err := s.AddCronJobOnce("", "* * * * *", func() {}); err == nil {
		t.Fatalf("expected AddCronJobOnce to fail without key")
	}

	cj, shutdown, err := cronjob.New()
	if err != nil {
		t.Fatalf("new cron scheduler: %v", err)
	}
	defer func() { _ = shutdown() }()

	s.cj = cj
	if err := s.AddCronJob("* * * * *", func() {}); err != nil {
		t.Fatalf("expected AddCronJob success, got %v", err)
	}
	if err := s.AddCronJobOnce("job1", "* * * * *", func() {}); err != nil {
		t.Fatalf("expected AddCronJobOnce first registration success, got %v", err)
	}
	if err := s.AddCronJobOnce("job1", "* * * * *", func() {}); err != nil {
		t.Fatalf("expected AddCronJobOnce duplicate no-op success, got %v", err)
	}

	ran := false
	if err := s.SendToWorker(func() { ran = true }); err != nil {
		t.Fatalf("expected nil workpool fallback to succeed: %v", err)
	}
	if !ran {
		t.Fatalf("expected fallback worker function to run synchronously")
	}

	wp, err := ants.NewPool(1)
	if err != nil {
		t.Fatalf("create ants pool: %v", err)
	}
	defer wp.Release()

	s.wp = wp
	done := make(chan struct{})
	if err := s.SendToWorker(func() { close(done) }); err != nil {
		t.Fatalf("expected workpool submit success: %v", err)
	}

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected submitted worker function to execute")
	}

	wp.Release()
	if err := s.SendToWorker(func() {}); err == nil {
		t.Fatalf("expected worker submit error after pool release")
	}
}

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

}
