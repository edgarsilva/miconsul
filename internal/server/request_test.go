package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

func TestCurrentUser(t *testing.T) {
	s := &Server{}

	runWithCtx(t, http.MethodGet, "/", nil, func(c fiber.Ctx) {
		if got := s.CurrentUser(c); got.ID != 0 || got.Email != "" {
			t.Fatalf("expected zero user when local is missing, got %#v", got)
		}
	})

	runWithCtx(t, http.MethodGet, "/", nil, func(c fiber.Ctx) {
		expected := models.User{UID: "usr_1", Email: "u@example.com"}
		c.Locals("current_user", expected)
		if got := s.CurrentUser(c); got.UID != expected.UID || got.Email != expected.Email {
			t.Fatalf("expected %#v, got %#v", expected, got)
		}
	})
}

func TestHTMXHelpers(t *testing.T) {
	s := &Server{}

	runWithCtx(t, http.MethodGet, "/", nil, func(c fiber.Ctx) {
		if s.IsHTMX(c) {
			t.Fatalf("expected request without HX-Request to be false")
		}
		if !s.NotHTMX(c) {
			t.Fatalf("expected request without HX-Request to be NotHTMX")
		}
	})

	runWithCtx(t, http.MethodGet, "/", map[string]string{"HX-Request": "true"}, func(c fiber.Ctx) {
		if !s.IsHTMX(c) {
			t.Fatalf("expected request with HX-Request=true to be HTMX")
		}
		if s.NotHTMX(c) {
			t.Fatalf("expected request with HX-Request=true to not be NotHTMX")
		}
	})
}

func TestRedirect(t *testing.T) {
	s := &Server{}
	app := fiber.New()
	app.Get("/from", func(c fiber.Ctx) error {
		return s.Redirect(c, "/to")
	})

	req := httptest.NewRequest(http.MethodGet, "/from", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusSeeOther {
		t.Fatalf("expected 303, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "/to" {
		t.Fatalf("expected redirect to /to, got %q", got)
	}
}

func TestCurrentThemeAndLocale(t *testing.T) {
	s := &Server{}

	runWithCtx(t, http.MethodGet, "/", nil, func(c fiber.Ctx) {
		if got := s.CurrentTheme(c); got != "light" {
			t.Fatalf("expected default theme light, got %q", got)
		}
		if got := s.CurrentLocale(c); got != "es-MX" {
			t.Fatalf("expected default locale es-MX, got %q", got)
		}

		c.Locals("theme", "dark")
		c.Locals("locale", "en-US")
		if got := s.CurrentTheme(c); got != "dark" {
			t.Fatalf("expected theme from locals, got %q", got)
		}
		if got := s.CurrentLocale(c); got != "en-US" {
			t.Fatalf("expected locale from locals, got %q", got)
		}
	})
}

func TestCurrentLocaleDoesNotWriteSession(t *testing.T) {
	s := &Server{SessionStore: session.NewStore()}
	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		c.Locals("locale", "en-US")
		_ = s.CurrentLocale(c)
		_ = s.L(c, "btn.save")
		return c.SendStatus(http.StatusNoContent)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil || resp.StatusCode != http.StatusNoContent {
		t.Fatalf("request failed status=%d err=%v", resp.StatusCode, err)
	}
	if cookies := resp.Header.Values("Set-Cookie"); len(cookies) > 0 {
		t.Fatalf("expected no session cookie from a locale read, got %v", cookies)
	}
}

func runWithCtx(t *testing.T, method, route string, headers map[string]string, fn func(c fiber.Ctx)) {
	t.Helper()

	app := fiber.New()
	app.Add([]string{method}, route, func(c fiber.Ctx) error {
		fn(c)
		return c.SendStatus(http.StatusNoContent)
	})

	req := httptest.NewRequest(method, route, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 helper response, got %d", resp.StatusCode)
	}
}
