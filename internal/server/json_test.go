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
	"miconsul/internal/models"

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

	t.Run("encodes nil slices as arrays and HTML unescaped", func(t *testing.T) {
		resp := doRequest(t, s, http.MethodGet, "/encode")
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		// Map key order is not guaranteed, so check each member on its own.
		for _, want := range []string{`"alpha":[]`, `"html":"<b>&</b>"`, `"zeta":1`} {
			if !strings.Contains(string(body), want) {
				t.Fatalf("body %s is missing %s", body, want)
			}
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

func TestUserSecretsSkippedByFiber(t *testing.T) {
	secrets := []string{"hashed-password", "reset-token-secret", "confirm-token-secret"}
	app := fiber.New(fiber.Config{JSONEncoder: marshalJSON, JSONDecoder: unmarshalJSON})
	app.Get("/user", func(c fiber.Ctx) error {
		return c.JSON(models.User{
			UID:               "user_123",
			Password:          "hashed-password",
			ResetToken:        "reset-token-secret",
			ConfirmEmailToken: "confirm-token-secret",
		})
	})
	app.Post("/user", func(c fiber.Ctx) error {
		var u models.User
		if err := c.Bind().Body(&u); err != nil {
			return err
		}
		if u.Name != "Ana" {
			t.Errorf("%s: expected Name to bind, got %q", c.Get(fiber.HeaderContentType), u.Name)
		}
		if u.Password != "" || u.ResetToken != "" || u.ConfirmEmailToken != "" {
			t.Errorf("%s: secret fields bound: Password=%q ResetToken=%q ConfirmEmailToken=%q",
				c.Get(fiber.HeaderContentType), u.Password, u.ResetToken, u.ConfirmEmailToken)
		}
		return c.SendStatus(fiber.StatusNoContent)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/user", nil))
	if err != nil {
		t.Fatalf("GET /user: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "user_123") {
		t.Fatalf("expected UID in response: %s", body)
	}
	for _, secret := range secrets {
		if strings.Contains(string(body), secret) {
			t.Errorf("c.JSON leaks %q: %s", secret, body)
		}
	}

	for _, tc := range []struct{ contentType, body string }{
		{fiber.MIMEApplicationJSON, `{"Name":"Ana","Password":"hashed-password","ResetToken":"reset-token-secret","ConfirmEmailToken":"confirm-token-secret"}`},
		{fiber.MIMEApplicationForm, "Name=Ana&Password=hashed-password&ResetToken=reset-token-secret&ConfirmEmailToken=confirm-token-secret"},
		{fiber.MIMEApplicationForm, "name=Ana&password=hashed-password&resettoken=reset-token-secret&confirmemailtoken=confirm-token-secret"},
	} {
		req := httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(tc.body))
		req.Header.Set(fiber.HeaderContentType, tc.contentType)
		resp, err := app.Test(req)
		if err != nil || resp.StatusCode != fiber.StatusNoContent {
			t.Fatalf("POST /user %s: status=%d err=%v", tc.contentType, resp.StatusCode, err)
		}
	}
}
