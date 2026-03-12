package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"
	"miconsul/internal/lib/cronjob"
	"miconsul/internal/lib/localize"
	"miconsul/internal/lib/workpool"
	"miconsul/internal/observability/logging"
	"miconsul/internal/observability/metrics"
	"miconsul/internal/server"

	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestIsExpectedServerCloseError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil error", err: nil, want: true},
		{name: "context canceled", err: context.Canceled, want: true},
		{name: "http server closed", err: http.ErrServerClosed, want: true},
		{name: "wrapped http server closed", err: fmt.Errorf("wrapped: %w", http.ErrServerClosed), want: true},
		{name: "network closed", err: net.ErrClosed, want: true},
		{name: "message contains server closed", err: errors.New("server closed unexpectedly"), want: true},
		{name: "message contains listener closed", err: errors.New("listener closed"), want: true},
		{name: "message contains closed network connection", err: errors.New("use of closed network connection"), want: true},
		{name: "unexpected error", err: errors.New("boom"), want: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isExpectedServerCloseError(tc.err)
			if got != tc.want {
				t.Fatalf("isExpectedServerCloseError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestShouldLogServerError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil error", err: nil, want: false},
		{name: "context canceled", err: context.Canceled, want: false},
		{name: "http server closed", err: http.ErrServerClosed, want: false},
		{name: "unexpected error", err: errors.New("boom"), want: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := shouldLogServerError(tc.err)
			if got != tc.want {
				t.Fatalf("shouldLogServerError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestSetupEnv(t *testing.T) {
	t.Run("loads env from process and returns parsed config", func(t *testing.T) {
		useTempWorkingDir(t)
		setRequiredEnv(t, map[string]string{})

		env, err := setupEnv()
		if err != nil {
			t.Fatalf("setupEnv() unexpected error: %v", err)
		}
		if env == nil {
			t.Fatal("setupEnv() expected non-nil env")
		}
		if env.AppName != "miconsul-test" {
			t.Fatalf("setupEnv() AppName = %q, want %q", env.AppName, "miconsul-test")
		}
	})

	t.Run("returns error when required env is missing", func(t *testing.T) {
		useTempWorkingDir(t)
		setRequiredEnv(t, map[string]string{"APP_NAME": ""})

		env, err := setupEnv()
		if err == nil {
			t.Fatal("setupEnv() expected error")
		}
		if env != nil {
			t.Fatal("setupEnv() expected nil env on error")
		}
	})
}

func TestSetupTelemetry(t *testing.T) {
	t.Run("builds telemetry runtime with real constructors", func(t *testing.T) {
		env := &appenv.Env{
			Environment:      appenv.EnvironmentDevelopment,
			AppName:          "miconsul-test",
			AppVersion:       "test",
			OTelTracerServer: "miconsul.server",
		}

		telemetry, err := setupTelemetry(context.Background(), env)
		if err != nil {
			t.Fatalf("setupTelemetry() unexpected error: %v", err)
		}
		if telemetry.tracer == nil {
			t.Fatal("setupTelemetry() expected tracer")
		}
		if telemetry.shutdownTracer == nil || telemetry.shutdownMetrics == nil || telemetry.shutdownLogs == nil {
			t.Fatal("setupTelemetry() expected shutdown callbacks")
		}
		if err := telemetry.shutdownTracer(); err != nil {
			t.Fatalf("setupTelemetry() shutdown tracer error: %v", err)
		}
		if err := telemetry.shutdownMetrics(); err != nil {
			t.Fatalf("setupTelemetry() shutdown metrics error: %v", err)
		}
		if err := telemetry.shutdownLogs(); err != nil {
			t.Fatalf("setupTelemetry() shutdown logs error: %v", err)
		}
	})

	t.Run("returns error for nil env", func(t *testing.T) {
		_, err := setupTelemetry(context.Background(), nil)
		if err == nil {
			t.Fatal("setupTelemetry() expected error")
		}
	})
}

func TestSetupDB(t *testing.T) {
	t.Run("creates sqlite db and applies migrations", func(t *testing.T) {
		env := &appenv.Env{DBPath: filepath.Join(t.TempDir(), "app.sqlite")}

		db, err := setupDB(env, logging.Logger{})
		if err != nil {
			if strings.Contains(err.Error(), "no such module: fts5") {
				return
			}
			t.Fatalf("setupDB() unexpected error: %v", err)
		}
		t.Cleanup(func() {
			_ = db.Close()
		})

		sqlDB, err := db.SQLDB()
		if err != nil {
			t.Fatalf("setupDB() SQLDB error: %v", err)
		}
		if err := sqlDB.Ping(); err != nil {
			t.Fatalf("setupDB() ping error: %v", err)
		}

		var tableCount int
		if err := db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'goose_db_version'").Scan(&tableCount).Error; err != nil {
			t.Fatalf("setupDB() verify goose table failed: %v", err)
		}
		if tableCount != 1 {
			t.Fatalf("setupDB() expected goose_db_version table, got count %d", tableCount)
		}
	})

	t.Run("returns error for nil env", func(t *testing.T) {
		_, err := setupDB(nil, logging.Logger{})
		if err == nil {
			t.Fatal("setupDB() expected error")
		}
	})
}

func TestSetupServer(t *testing.T) {
	setWorkingDirToRepoRoot(t)

	gormDB, err := gorm.Open(sqlite.Open("file:cmd_app_setup_server?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	wp, shutdownWorkPool := workpool.New(1)
	t.Cleanup(shutdownWorkPool)

	localizer := localize.New("en-US", "es-MX")
	telemetry := telemetryRuntime{
		tracer:        noop.NewTracerProvider().Tracer("test"),
		httpMetrics:   metrics.HTTPMetrics{},
		requestLogger: logging.Logger{},
	}

	s := setupServer(
		&appenv.Env{Environment: appenv.EnvironmentDevelopment, CookieSecret: "0123456789abcdef0123456789abcdef", AppProtocol: "http"},
		&database.Database{DB: gormDB},
		&cronjob.Sched{},
		wp,
		telemetry,
		localizer,
	)

	if s == nil {
		t.Fatal("setupServer() expected non-nil server")
	}
	if s.App == nil {
		t.Fatal("setupServer() expected non-nil app")
	}
	if s.SessionStore == nil {
		t.Fatal("setupServer() expected non-nil session store")
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	resp, err := s.App.Test(req)
	if err != nil {
		t.Fatalf("setupServer() metrics request failed: %v", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		t.Fatalf("setupServer() expected /metrics to be registered, got status %d", resp.StatusCode)
	}
}

func TestRunServerLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("does not shutdown when listen exits immediately", func(t *testing.T) {
		t.Parallel()

		runner := newFakeRunner(errors.New("listen failed"))

		done := make(chan struct{})
		go func() {
			runServerLifecycle(context.Background(), runner, 8080, 200*time.Millisecond)
			close(done)
		}()

		runner.unblockListen()
		<-done

		if runner.shutdownCalled() {
			t.Fatal("runServerLifecycle() expected no shutdown call")
		}
	})

	t.Run("attempts graceful shutdown when context is canceled", func(t *testing.T) {
		t.Parallel()

		runner := newFakeRunner(net.ErrClosed)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		runServerLifecycle(ctx, runner, 8080, 200*time.Millisecond)

		if !runner.shutdownCalled() {
			t.Fatal("runServerLifecycle() expected shutdown call")
		}
	})
}

func TestMainSuccessPath(t *testing.T) {
	restore := installMainTestHooks()
	defer restore()

	lifecycleCalled := false
	shutdownTracerCalled := false
	shutdownMetricsCalled := false
	shutdownLogsCalled := false
	shutdownCronCalled := false
	shutdownWorkPoolCalled := false
	exitCode := 0

	setupEnvForMain = func() (*appenv.Env, error) {
		return &appenv.Env{AppPort: 3000, AppShutdownTimeout: 10 * time.Millisecond}, nil
	}
	setupTelemetryForMain = func(context.Context, *appenv.Env) (telemetryRuntime, error) {
		return telemetryRuntime{
			shutdownTracer:  func() error { shutdownTracerCalled = true; return nil },
			shutdownMetrics: func() error { shutdownMetricsCalled = true; return nil },
			shutdownLogs:    func() error { shutdownLogsCalled = true; return nil },
		}, nil
	}
	setupDBForMain = func(*appenv.Env, logging.Logger) (*database.Database, error) {
		return &database.Database{}, nil
	}
	newCronjobForMain = func() (*cronjob.Sched, func() error, error) {
		return &cronjob.Sched{}, func() error { shutdownCronCalled = true; return nil }, nil
	}
	newWorkpoolForMain = func(int) (*workpool.Pool, func()) {
		return &workpool.Pool{}, func() { shutdownWorkPoolCalled = true }
	}
	setupServerForMain = func(*appenv.Env, *database.Database, *cronjob.Sched, *workpool.Pool, telemetryRuntime, *localize.Localizer) *server.Server {
		return &server.Server{}
	}
	registerRoutesForMain = func(*server.Server) error { return nil }
	runLifecycleForMain = func(context.Context, serverRunner, int, time.Duration) { lifecycleCalled = true }
	notifyContextForMain = func(context.Context, ...os.Signal) (context.Context, context.CancelFunc) {
		return context.WithCancel(context.Background())
	}
	exitForMain = func(code int) { exitCode = code }

	main()

	if exitCode != 0 {
		t.Fatalf("main() exit code = %d, want 0", exitCode)
	}
	if !lifecycleCalled {
		t.Fatal("main() expected run lifecycle call")
	}
	if !shutdownTracerCalled || !shutdownMetricsCalled || !shutdownLogsCalled {
		t.Fatalf("main() expected telemetry shutdown calls, got tracer=%v metrics=%v logs=%v", shutdownTracerCalled, shutdownMetricsCalled, shutdownLogsCalled)
	}
	if !shutdownCronCalled {
		t.Fatal("main() expected cron shutdown call")
	}
	if !shutdownWorkPoolCalled {
		t.Fatal("main() expected workpool shutdown call")
	}
}

func TestMainSetupEnvFailure(t *testing.T) {
	restore := installMainTestHooks()
	defer restore()

	exitCode := 0
	setupEnvForMain = func() (*appenv.Env, error) { return nil, errors.New("env failed") }
	exitForMain = func(code int) { exitCode = code }

	main()

	if exitCode != 1 {
		t.Fatalf("main() exit code = %d, want 1", exitCode)
	}
}

func TestMainRegisterRoutesFailure(t *testing.T) {
	restore := installMainTestHooks()
	defer restore()

	exitCode := 0
	setupEnvForMain = func() (*appenv.Env, error) {
		return &appenv.Env{AppPort: 3000, AppShutdownTimeout: 10 * time.Millisecond}, nil
	}
	setupTelemetryForMain = func(context.Context, *appenv.Env) (telemetryRuntime, error) {
		return telemetryRuntime{
			shutdownTracer:  func() error { return nil },
			shutdownMetrics: func() error { return nil },
			shutdownLogs:    func() error { return nil },
		}, nil
	}
	setupDBForMain = func(*appenv.Env, logging.Logger) (*database.Database, error) {
		return &database.Database{}, nil
	}
	newCronjobForMain = func() (*cronjob.Sched, func() error, error) {
		return &cronjob.Sched{}, func() error { return nil }, nil
	}
	newWorkpoolForMain = func(int) (*workpool.Pool, func()) {
		return &workpool.Pool{}, func() {}
	}
	setupServerForMain = func(*appenv.Env, *database.Database, *cronjob.Sched, *workpool.Pool, telemetryRuntime, *localize.Localizer) *server.Server {
		return &server.Server{}
	}
	registerRoutesForMain = func(*server.Server) error { return errors.New("routes failed") }
	runLifecycleForMain = func(context.Context, serverRunner, int, time.Duration) {
		t.Fatal("main() should not run lifecycle when route registration fails")
	}
	notifyContextForMain = func(context.Context, ...os.Signal) (context.Context, context.CancelFunc) {
		return context.WithCancel(context.Background())
	}
	exitForMain = func(code int) { exitCode = code }

	main()

	if exitCode != 1 {
		t.Fatalf("main() exit code = %d, want 1", exitCode)
	}
}

type fakeRunner struct {
	listenErr error

	startedListen chan struct{}
	listenBlock   chan struct{}

	shutdownOnce sync.Once
	shutdownHits atomic.Int32
}

func newFakeRunner(listenErr error) *fakeRunner {
	return &fakeRunner{
		listenErr:     listenErr,
		startedListen: make(chan struct{}),
		listenBlock:   make(chan struct{}),
	}
}

func (r *fakeRunner) Listen(_ ...int) error {
	close(r.startedListen)
	<-r.listenBlock
	return r.listenErr
}

func (r *fakeRunner) ShutdownWithContext(context.Context) error {
	r.shutdownHits.Add(1)
	r.shutdownOnce.Do(func() {
		close(r.listenBlock)
	})

	return nil
}

func (r *fakeRunner) unblockListen() {
	<-r.startedListen
	r.shutdownOnce.Do(func() {
		close(r.listenBlock)
	})
}

func (r *fakeRunner) shutdownCalled() bool {
	return r.shutdownHits.Load() > 0
}

func useTempWorkingDir(t *testing.T) {
	t.Helper()

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Fatalf("restore working dir: %v", err)
		}
	})
}

func setRequiredEnv(t *testing.T, overrides map[string]string) {
	t.Helper()

	values := map[string]string{
		"APP_ENV":             "development",
		"APP_NAME":            "miconsul-test",
		"APP_PROTOCOL":        "http",
		"APP_DOMAIN":          "localhost",
		"APP_VERSION":         "test",
		"APP_PORT":            "3000",
		"COOKIE_SECRET":       "0123456789abcdef0123456789abcdef",
		"JWT_SECRET":          "jwt-test-secret",
		"DB_PATH":             filepath.Join(t.TempDir(), "env_app.sqlite"),
		"SESSION_DB_PATH":     filepath.Join(t.TempDir(), "env_session.sqlite"),
		"EMAIL_SENDER":        "test",
		"EMAIL_SECRET":        "test",
		"EMAIL_FROM_ADDRESS":  "test@example.com",
		"EMAIL_SMTP_URL":      "smtp://localhost:1025",
		"GOOSE_DRIVER":        "sqlite3",
		"GOOSE_DBSTRING":      "file:test.sqlite",
		"GOOSE_MIGRATION_DIR": ".",
		"ASSETS_DIR":          "assets",
	}

	for key, value := range overrides {
		values[key] = value
	}

	for key, value := range values {
		t.Setenv(key, value)
	}
}

func setWorkingDirToRepoRoot(t *testing.T) {
	t.Helper()

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	root, err := findRepoRoot(originalWD)
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir repo root: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})
}

func findRepoRoot(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}

		dir = parent
	}
}

func installMainTestHooks() func() {
	oldSetupEnv := setupEnvForMain
	oldSetupTelemetry := setupTelemetryForMain
	oldSetupDB := setupDBForMain
	oldNewCronjob := newCronjobForMain
	oldNewWorkpool := newWorkpoolForMain
	oldSetupServer := setupServerForMain
	oldRegisterRoutes := registerRoutesForMain
	oldRunLifecycle := runLifecycleForMain
	oldExit := exitForMain
	oldNotifyContext := notifyContextForMain

	return func() {
		setupEnvForMain = oldSetupEnv
		setupTelemetryForMain = oldSetupTelemetry
		setupDBForMain = oldSetupDB
		newCronjobForMain = oldNewCronjob
		newWorkpoolForMain = oldNewWorkpool
		setupServerForMain = oldSetupServer
		registerRoutesForMain = oldRegisterRoutes
		runLifecycleForMain = oldRunLifecycle
		exitForMain = oldExit
		notifyContextForMain = oldNotifyContext
	}
}
