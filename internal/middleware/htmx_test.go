package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestHTMXOnly(t *testing.T) {
	tests := []struct {
		name      string
		hxRequest string
		want      int
	}{
		{"htmx request reaches the handler", "true", http.StatusNoContent},
		{"plain request gets 404", "", http.StatusNotFound},
		{"other header value gets 404", "false", http.StatusNotFound},
	}

	for _, tt := range tests {
		app := fiber.New()
		app.Get("/", HTMXOnly(), func(c fiber.Ctx) error {
			return c.SendStatus(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if tt.hxRequest != "" {
			req.Header.Set("HX-Request", tt.hxRequest)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("%s: request failed: %v", tt.name, err)
		}
		if resp.StatusCode != tt.want {
			t.Errorf("%s: status = %d, want %d", tt.name, resp.StatusCode, tt.want)
		}
	}
}
