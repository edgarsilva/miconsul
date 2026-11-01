package clinic

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
)

func TestClinicHandlerBranches(t *testing.T) {
	svc, user := newClinicServiceForTests(t)

	app := fiber.New()
	app.Get("/clinics/search", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleIndexSearch(c)
	})
	app.Get("/clinics/:id", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleShowPage(c)
	})
	app.Post("/clinics", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleCreate(c)
	})
	app.Post("/clinics/:id/patch", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleUpdate(c)
	})
	app.Post("/clinics/:id/delete", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleDelete(c)
	})

	resp1, err := app.Test(httptest.NewRequest(http.MethodGet, "/clinics/%20", nil))
	if err != nil || resp1.StatusCode != fiber.StatusNotFound {
		t.Fatalf("show with missing clinic expected 404, got status=%d err=%v", resp1.StatusCode, err)
	}

	resp2, err := app.Test(httptest.NewRequest(http.MethodGet, "/clinics/search?searchTerm=ab", nil))
	if err != nil || resp2.StatusCode != fiber.StatusOK {
		t.Fatalf("search short term expected 200 toast, got status=%d err=%v", resp2.StatusCode, err)
	}

	badJSON := httptest.NewRequest(http.MethodPost, "/clinics", strings.NewReader("{"))
	badJSON.Header.Set("Content-Type", "application/json")
	badJSON.Header.Set("HX-Request", "true")
	resp3, err := app.Test(badJSON)
	if err != nil || resp3.StatusCode != fiber.StatusNoContent {
		t.Fatalf("create invalid body expected 204 for htmx redirect, got status=%d err=%v", resp3.StatusCode, err)
	}

	resp4, err := app.Test(httptest.NewRequest(http.MethodPost, "/clinics/%20/patch", nil))
	if err != nil || resp4.StatusCode != fiber.StatusNotFound {
		t.Fatalf("update missing clinic expected 404, got status=%d err=%v", resp4.StatusCode, err)
	}

	resp5, err := app.Test(httptest.NewRequest(http.MethodPost, "/clinics/%20/delete", nil))
	if err != nil || resp5.StatusCode != fiber.StatusNotFound {
		t.Fatalf("delete missing clinic expected 404, got status=%d err=%v", resp5.StatusCode, err)
	}
}

func TestClinicHandlersHappyPaths(t *testing.T) {
	svc, user := newClinicServiceForTests(t)

	clinic := models.Clinic{UserID: user.ID, Name: "Alpha", Email: "alpha@example.com", Phone: "123"}
	if err := svc.CreateClinic(t.Context(), &clinic); err != nil {
		t.Fatalf("seed clinic: %v", err)
	}

	app := fiber.New()
	app.Get("/clinics/:id", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleShowPage(c)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/clinics/"+clinic.UID, nil))
	if err != nil || resp.StatusCode != fiber.StatusOK {
		t.Fatalf("show existing clinic expected 200, got status=%d err=%v", resp.StatusCode, err)
	}
}

func TestIndexSearchToasts(t *testing.T) {
	tests := []struct {
		name       string
		term       string
		htmx       bool
		wantStatus int
		wantToast  string
	}{
		{"short term", "ab", true, fiber.StatusOK, "at least 3 characters"},
		// The test DB has no FTS table, so any real search term fails.
		{"search error", "alpha", true, fiber.StatusInternalServerError, "couldn&#39;t be completed"},
		{"short term without htmx", "ab", false, fiber.StatusOK, "at least 3 characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, user := newClinicServiceForTests(t)

			app := fiber.New()
			app.Get("/clinics/search", func(c fiber.Ctx) error {
				c.Locals("current_user", user)
				return svc.HandleIndexSearch(c)
			})

			req := httptest.NewRequest(http.MethodGet, "/clinics/search?searchTerm="+tt.term, nil)
			if tt.htmx {
				req.Header.Set("HX-Request", "true")
			}
			resp, err := app.Test(req)
			if err != nil || resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got status=%d err=%v", tt.wantStatus, resp.StatusCode, err)
			}
			if got := resp.Header.Get("HX-Redirect"); got != "" {
				t.Errorf("search redirected to %q", got)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if !strings.Contains(string(body), `<hx-partial hx-target="#toast_region"`) || !strings.Contains(string(body), tt.wantToast) {
				t.Errorf("body = %s, want only a toast partial with %q", body, tt.wantToast)
			}
			if strings.Contains(string(body), `id="clinics_list"`) {
				t.Errorf("search replaced the list")
			}
		})
	}
}
