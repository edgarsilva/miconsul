package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"

	"github.com/gofiber/fiber/v3"
)

func TestFiberJSONUsesV2(t *testing.T) {
	setWorkingDirToRepoRoot(t)

	s := &Server{
		Env: &appenv.Env{
			Environment:   appenv.EnvironmentDevelopment,
			AppProtocol:   "http",
			CookieSecret:  "0123456789abcdef0123456789abcdef",
			SessionDBPath: filepath.Join(t.TempDir(), "session.sqlite"),
		},
		DB: &database.Database{DB: openTestDB(t)},
	}
	setupSessionStore(s)
	setupFiberApp(s)

	type signin struct {
		Email      string `json:"email"`
		RememberMe bool   `json:"remember_me"`
	}

	s.Post("/bind", func(c fiber.Ctx) error {
		var in signin
		if err := c.Bind().Body(&in); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString(err.Error())
		}
		return c.JSON(in)
	})
	s.Get("/encode", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"zeta":  1,
			"alpha": []string(nil),
			"html":  "<b>&</b>",
		})
	})

	t.Run("decodes tagged field names", func(t *testing.T) {
		got := postJSON(t, s, `{"email":"a@b.co","remember_me":true}`)
		if got != `{"email":"a@b.co","remember_me":true}` {
			t.Fatalf("unexpected body %s", got)
		}
	})

	t.Run("matches field names case-sensitively", func(t *testing.T) {
		got := postJSON(t, s, `{"Email":"a@b.co"}`)
		if got != `{"email":"","remember_me":false}` {
			t.Fatalf("expected mixed-case key to be ignored, got %s", got)
		}
	})

	t.Run("encodes sorted keys, nil slices as arrays, unescaped HTML", func(t *testing.T) {
		resp := doRequest(t, s, http.MethodGet, "/encode")
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		want := `{"alpha":[],"html":"<b>&</b>","zeta":1}`
		if string(body) != want {
			t.Fatalf("body = %s, want %s", body, want)
		}
	})
}

func postJSON(t *testing.T, s *Server, body string) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/bind", strings.NewReader(body))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

	resp, err := s.App.Test(req)
	if err != nil {
		t.Fatalf("POST /bind failed: %v", err)
	}

	out, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("POST /bind status = %d, body %s", resp.StatusCode, out)
	}

	return string(out)
}
