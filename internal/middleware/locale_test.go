package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestLocaleLang(t *testing.T) {
	tests := []struct {
		name           string
		cookie         string
		acceptLanguage string
		want           string
	}{
		{"nothing uses the default", "", "", "es-MX"},
		{"any language uses the default", "", "*", "es-MX"},
		{"english browser with spanish too", "", "en-US,en;q=0.9,es;q=0.8", "en-US"},
		{"english browser", "", "en-GB,en;q=0.9", "en-US"},
		{"spanish browser", "", "es-ES,es;q=0.9", "es-MX"},
		{"unsupported browser language uses the default", "", "fr-CA", "es-MX"},
		{"cookie beats the browser", "en-US", "es-MX", "en-US"},
		{"unknown cookie falls back to the browser", "fr-CA", "en-US", "en-US"},
	}

	for _, tt := range tests {
		app := fiber.New()
		app.Use(LocaleLang())
		app.Get("/", func(c fiber.Ctx) error {
			if got, _ := c.Locals("locale").(string); got != tt.want {
				t.Errorf("%s: locale = %q, want %q", tt.name, got, tt.want)
			}
			return c.SendStatus(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if tt.cookie != "" {
			req.AddCookie(&http.Cookie{Name: "locale", Value: tt.cookie})
		}
		if tt.acceptLanguage != "" {
			req.Header.Set("Accept-Language", tt.acceptLanguage)
		}
		resp, err := app.Test(req)
		if err != nil || resp.StatusCode != http.StatusNoContent {
			t.Fatalf("%s: status=%d err=%v", tt.name, resp.StatusCode, err)
		}
	}
}
