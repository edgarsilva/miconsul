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
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

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
