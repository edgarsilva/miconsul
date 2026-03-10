package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/model"

	"github.com/gofiber/fiber/v3"
)

func TestCurrentUser(t *testing.T) {
	s := &Server{}

	runWithCtx(t, http.MethodGet, "/", nil, func(c fiber.Ctx) {
		if got := s.CurrentUser(c); got.ID != "" || got.Email != "" {
			t.Fatalf("expected zero user when local is missing, got %#v", got)
		}
	})

	runWithCtx(t, http.MethodGet, "/", nil, func(c fiber.Ctx) {
		expected := model.User{ID: "usr_1", Email: "u@example.com"}
		c.Locals("current_user", expected)
		if got := s.CurrentUser(c); got.ID != expected.ID || got.Email != expected.Email {
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

func TestNewCookie(t *testing.T) {
	t.Run("development defaults to insecure", func(t *testing.T) {
		s := &Server{Env: &appenv.Env{Environment: appenv.EnvironmentDevelopment, AppProtocol: "http"}}
		cookie := s.NewCookie("theme", "dark", time.Hour)
		if cookie.Secure {
			t.Fatalf("expected secure=false for development http")
		}
		if !cookie.HTTPOnly {
			t.Fatalf("expected HTTPOnly=true")
		}
	})

	t.Run("production enables secure", func(t *testing.T) {
		s := &Server{Env: &appenv.Env{Environment: appenv.EnvironmentProduction, AppProtocol: "http"}}
		cookie := s.NewCookie("theme", "dark", time.Hour)
		if !cookie.Secure {
			t.Fatalf("expected secure=true for production")
		}
	})

	t.Run("https protocol enables secure", func(t *testing.T) {
		s := &Server{Env: &appenv.Env{Environment: appenv.EnvironmentDevelopment, AppProtocol: "https"}}
		cookie := s.NewCookie("theme", "dark", time.Hour)
		if !cookie.Secure {
			t.Fatalf("expected secure=true for https protocol")
		}
	})
}

func TestInvalidateCookies(t *testing.T) {
	s := &Server{Env: &appenv.Env{Environment: appenv.EnvironmentDevelopment, AppProtocol: "http"}}
	app := fiber.New()
	app.Get("/logout", func(c fiber.Ctx) error {
		s.InvalidateCookies(c, "auth", "theme")
		return c.SendStatus(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	cookies := strings.Join(resp.Header.Values("Set-Cookie"), ";")
	if !strings.Contains(cookies, "auth=") {
		t.Fatalf("expected auth cookie invalidation header, got %q", cookies)
	}
	if !strings.Contains(cookies, "theme=") {
		t.Fatalf("expected theme cookie invalidation header, got %q", cookies)
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
