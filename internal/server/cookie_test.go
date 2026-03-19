package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"miconsul/internal/lib/appenv"

	"github.com/gofiber/fiber/v3"
)

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
