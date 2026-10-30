package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"miconsul/internal/server/servertest"

	"github.com/gofiber/fiber/v3"
)

func TestRedirectWithToast(t *testing.T) {
	tests := []struct {
		name         string
		htmx         bool
		wantStatus   int
		wantLocation string
		wantHXRedir  string
	}{
		{"regular request gets a 303", false, http.StatusSeeOther, "/clinics", ""},
		{"htmx request gets HX-Redirect", true, http.StatusNoContent, "/clinics", "/clinics"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{}
			app := fiber.New()
			app.Post("/", func(c fiber.Ctx) error {
				return s.RedirectWithToast(c, "/clinics", "Clinic saved", "success")
			})

			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if tt.htmx {
				req.Header.Set("HX-Request", "true")
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			if got := resp.Header.Get("Location"); got != tt.wantLocation {
				t.Errorf("Location = %q, want %q", got, tt.wantLocation)
			}
			if got := resp.Header.Get("HX-Redirect"); got != tt.wantHXRedir {
				t.Errorf("HX-Redirect = %q, want %q", got, tt.wantHXRedir)
			}
			if msg, level := servertest.FlashToast(t, resp); msg != "Clinic saved" || level != "success" {
				t.Errorf("flash toast = %q %q, want Clinic saved success", msg, level)
			}
		})
	}
}
