package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"

	fiberhealthcheck "github.com/gofiber/fiber/v3/middleware/healthcheck"
	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestValidateCriticalDeps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    *Server
		want string
	}{
		{name: "nil server", s: nil, want: "server is required"},
		{name: "missing env", s: &Server{}, want: "environment config is required"},
		{name: "missing db", s: &Server{Env: &appenv.Env{}}, want: "Database is required"},
		{name: "missing tracer", s: &Server{Env: &appenv.Env{}, DB: &database.Database{}}, want: "tracer is required; pass server.WithTracer(...) to server.New(...)"},
		{
			name: "valid deps",
			s: &Server{
				Env:    &appenv.Env{},
				DB:     &database.Database{},
				Tracer: noop.NewTracerProvider().Tracer("test"),
			},
			want: "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateCriticalDeps(tc.s)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("validateCriticalDeps() unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("validateCriticalDeps() expected error %q, got nil", tc.want)
			}
			if err.Error() != tc.want {
				t.Fatalf("validateCriticalDeps() error = %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

func TestValidateRuntimeConfig(t *testing.T) {
	t.Parallel()

	validCookieSecret := "0123456789abcdef0123456789abcdef"

	tests := []struct {
		name string
		s    *Server
		want string
	}{
		{name: "nil server", s: nil, want: "server is required"},
		{
			name: "invalid environment",
			s:    &Server{Env: &appenv.Env{Environment: appenv.Environment("not-valid"), CookieSecret: validCookieSecret}},
			want: "APP_ENV is invalid",
		},
		{
			name: "missing cookie secret",
			s:    &Server{Env: &appenv.Env{Environment: appenv.EnvironmentDevelopment}},
			want: "COOKIE_SECRET is required",
		},
		{
			name: "short cookie secret",
			s:    &Server{Env: &appenv.Env{Environment: appenv.EnvironmentDevelopment, CookieSecret: "short"}},
			want: "COOKIE_SECRET must be at least 32 characters",
		},
		{
			name: "valid runtime config",
			s:    &Server{Env: &appenv.Env{Environment: appenv.EnvironmentDevelopment, CookieSecret: validCookieSecret}},
			want: "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateRuntimeConfig(tc.s)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("validateRuntimeConfig() unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("validateRuntimeConfig() expected error %q, got nil", tc.want)
			}
			if err.Error() != tc.want {
				t.Fatalf("validateRuntimeConfig() error = %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

func TestSetupSessionStoreAndFiberApp(t *testing.T) {
	setWorkingDirToRepoRoot(t)

	appDB := openTestDB(t)
	tmpDir := t.TempDir()

	s := &Server{
		Env: &appenv.Env{
			Environment:   appenv.EnvironmentDevelopment,
			AppProtocol:   "http",
			CookieSecret:  "0123456789abcdef0123456789abcdef",
			SessionDBPath: filepath.Join(tmpDir, "session.sqlite3"),
		},
		DB:                &database.Database{DB: appDB},
		StartedAt:         time.Now().Add(-3 * time.Second),
		ReadyAt:           time.Now().Add(-2 * time.Second),
		BootstrapDuration: time.Second,
	}

	setupSessionStore(s)
	if s.SessionStore == nil {
		t.Fatal("setupSessionStore() expected non-nil SessionStore")
	}

	setupFiberApp(s)
	if s.App == nil {
		t.Fatal("setupFiberApp() expected non-nil App")
	}

	assertRouteNot404(t, s, http.MethodGet, "/metrics")
	assertStatus(t, s, http.MethodGet, fiberhealthcheck.LivenessEndpoint, http.StatusOK)
	assertStatus(t, s, http.MethodGet, fiberhealthcheck.ReadinessEndpoint, http.StatusOK)
	assertStatus(t, s, http.MethodGet, fiberhealthcheck.StartupEndpoint, http.StatusOK)
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
		t.Fatalf("chdir to repo root: %v", err)
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

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:server_setup?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}

	return db
}

func assertRouteNot404(t *testing.T, s *Server, method, path string) {
	t.Helper()

	resp := doRequest(t, s, method, path)
	if resp.StatusCode == http.StatusNotFound {
		t.Fatalf("expected %s %s to be registered, got 404", method, path)
	}
}

func assertStatus(t *testing.T, s *Server, method, path string, wantStatus int) {
	t.Helper()

	resp := doRequest(t, s, method, path)
	if resp.StatusCode != wantStatus {
		t.Fatalf("%s %s status = %d, want %d", method, path, resp.StatusCode, wantStatus)
	}
}

func doRequest(t *testing.T, s *Server, method, path string) *http.Response {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	resp, err := s.App.Test(req)
	if err != nil {
		t.Fatalf("request %s %s failed: %v", method, path, err)
	}

	return resp
}
