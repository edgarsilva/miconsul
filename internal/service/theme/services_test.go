package theme

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"miconsul/internal/server"

	"github.com/gofiber/fiber/v3"
)

func TestNewService(t *testing.T) {
	t.Run("nil server fails", func(t *testing.T) {
		_, err := NewService(nil)
		if err == nil {
			t.Fatalf("expected nil server error")
		}
	})

	t.Run("non-nil server succeeds", func(t *testing.T) {
		svc, err := NewService(&server.Server{})
		if err != nil {
			t.Fatalf("unexpected NewService error: %v", err)
		}
		if svc.Server == nil {
			t.Fatalf("expected service to keep server reference")
		}
	})
}

func TestHandleToggleTheme(t *testing.T) {
	app := fiber.New()
	svc := service{Server: &server.Server{}}
	app.Post("/theme/toggle/:theme", svc.HandleToggleTheme)

	req := httptest.NewRequest(http.MethodPost, "/theme/toggle/dark", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	setCookie := resp.Header.Get("Set-Cookie")
	if setCookie == "" {
		t.Fatalf("expected Set-Cookie header")
	}
	if !strings.Contains(setCookie, "theme=dark") {
		t.Fatalf("expected theme cookie set to dark, got %q", setCookie)
	}
}
