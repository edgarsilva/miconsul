package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestUITheme(t *testing.T) {
	tests := []struct {
		name   string
		cookie string
		want   string
	}{
		{"no cookie follows the OS", "", ""},
		{"light", "light", "light"},
		{"dark", "dark", "dark"},
		{"unknown value follows the OS", "dracula", ""},
	}

	for _, tt := range tests {
		app := fiber.New()
		app.Use(UITheme())
		app.Get("/", func(c fiber.Ctx) error {
			if got, _ := c.Locals("theme").(string); got != tt.want {
				t.Errorf("%s: theme = %q, want %q", tt.name, got, tt.want)
			}
			return c.SendStatus(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if tt.cookie != "" {
			req.AddCookie(&http.Cookie{Name: "theme", Value: tt.cookie})
		}
		resp, err := app.Test(req)
		if err != nil || resp.StatusCode != http.StatusNoContent {
			t.Fatalf("%s: status=%d err=%v", tt.name, resp.StatusCode, err)
		}
	}
}
