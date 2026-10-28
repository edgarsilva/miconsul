package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"miconsul/internal/lib/appenv"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

func TestSessionConfigDefaults(t *testing.T) {
	t.Chdir(t.TempDir())

	cfg := sessionConfig("")
	if cfg.Database != defaultSessionDBPath {
		t.Fatalf("expected default session db path %q, got %q", defaultSessionDBPath, cfg.Database)
	}
	if cfg.Table != "fiber_storage" {
		t.Fatalf("expected default session table name, got %q", cfg.Table)
	}
	if _, err := os.Stat(filepath.Dir(defaultSessionDBPath)); err != nil {
		t.Fatalf("expected session db directory to be created: %v", err)
	}
}

func TestSessionConfigPaths(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "file path", path: filepath.Join(dir, "nested", "custom.sqlite"), want: filepath.Join(dir, "nested", "custom.sqlite")},
		{name: "directory path", path: dir, want: filepath.Join(dir, sessionDBFileName)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sessionConfig(tt.path).Database; got != tt.want {
				t.Fatalf("sessionConfig(%q).Database = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestSessionCookieAttributes(t *testing.T) {
	s := &Server{Env: &appenv.Env{
		Environment:   appenv.EnvironmentDevelopment,
		AppProtocol:   "http",
		SessionDBPath: filepath.Join(t.TempDir(), "session.sqlite3"),
	}}
	if err := setupSessionStore(s); err != nil {
		t.Fatalf("setup session store: %v", err)
	}

	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		if err := s.SessionWrite(c, "k", "v"); err != nil {
			return err
		}
		return c.SendStatus(http.StatusNoContent)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	for _, c := range resp.Cookies() {
		if c.Name != "session_id" {
			continue
		}
		if !c.HttpOnly {
			t.Fatalf("expected session_id to be HttpOnly")
		}
		if c.SameSite != http.SameSiteLaxMode {
			t.Fatalf("expected session_id SameSite=Lax, got %v", c.SameSite)
		}
		if got := time.Duration(c.MaxAge) * time.Second; got != sessionIdleTimeout {
			t.Fatalf("expected session_id Max-Age %v, got %v", sessionIdleTimeout, got)
		}
		return
	}

	t.Fatalf("expected a session_id cookie")
}

func TestSessionRotate(t *testing.T) {
	s := &Server{SessionStore: session.NewStore()}

	app := fiber.New()
	app.Post("/write", func(c fiber.Ctx) error {
		return s.SessionWrite(c, "before", "kept")
	})
	app.Post("/rotate", func(c fiber.Ctx) error {
		return s.SessionRotate(c, "after", "set")
	})

	write, err := app.Test(httptest.NewRequest(http.MethodPost, "/write", nil))
	if err != nil {
		t.Fatalf("write request failed: %v", err)
	}
	oldID := sessionIDCookie(write)

	req := httptest.NewRequest(http.MethodPost, "/rotate", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: oldID})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("rotate request failed: %v", err)
	}

	count := 0
	for _, c := range resp.Cookies() {
		if c.Name == "session_id" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one session_id cookie, got %d", count)
	}

	newID := sessionIDCookie(resp)
	if newID == "" || newID == oldID {
		t.Fatalf("expected a new session_id, got %q (was %q)", newID, oldID)
	}
	if data, err := s.SessionStore.Storage.Get(oldID); err != nil || data != nil {
		t.Fatalf("expected the old session deleted, got %q, %v", data, err)
	}

	sess, err := s.SessionStore.GetByID(t.Context(), newID)
	if err != nil {
		t.Fatalf("load rotated session: %v", err)
	}
	defer sess.Release()
	if got := sess.Get("before"); got != "kept" {
		t.Fatalf("expected earlier data kept, got %v", got)
	}
	if got := sess.Get("after"); got != "set" {
		t.Fatalf("expected the rotated value set, got %v", got)
	}
}

func sessionIDCookie(resp *http.Response) string {
	for _, c := range resp.Cookies() {
		if c.Name == "session_id" {
			return c.Value
		}
	}

	return ""
}
