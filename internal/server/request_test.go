package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"miconsul/internal/models"

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
		expected := models.User{ID: "usr_1", Email: "u@example.com"}
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
