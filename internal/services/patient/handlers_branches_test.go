package patient

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestPatientHandlerGuardBranches(t *testing.T) {
	svc, user := newPatientServiceForTests(t)

	app := fiber.New()
	app.Get("/patients/:id", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandlePatientFormPage(c)
	})
	app.Post("/patients", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleCreatePatient(c)
	})
	app.Post("/patients/:id/patch", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleUpdatePatient(c)
	})
	app.Post("/patients/:id/removepic", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleRemovePic(c)
	})
	app.Post("/patients/:id/delete", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleDeletePatient(c)
	})
	app.Get("/patients/search", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandlePatientsIndexSearch(c)
	})
	app.Get("/patients/:id/profilepic/:filename", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandlePatientProfilePicImgSrc(c)
	})

	resp1, err := app.Test(httptest.NewRequest(http.MethodGet, "/patients/%20", nil))
	if err != nil || resp1.StatusCode != fiber.StatusSeeOther {
		t.Fatalf("patient form blank id expected 303, got status=%d err=%v", resp1.StatusCode, err)
	}

	badJSON := httptest.NewRequest(http.MethodPost, "/patients", strings.NewReader("{"))
	badJSON.Header.Set("Content-Type", "application/json")
	resp2, err := app.Test(badJSON)
	if err != nil || resp2.StatusCode != fiber.StatusSeeOther {
		t.Fatalf("create invalid body expected 303 redirect, got status=%d err=%v", resp2.StatusCode, err)
	}

	resp3, err := app.Test(httptest.NewRequest(http.MethodPost, "/patients//patch", nil))
	if err != nil || resp3.StatusCode != fiber.StatusNotFound {
		t.Fatalf("patch missing route id expected 404, got status=%d err=%v", resp3.StatusCode, err)
	}

	resp4, err := app.Test(httptest.NewRequest(http.MethodPost, "/patients//removepic", nil))
	if err != nil || resp4.StatusCode != fiber.StatusNotFound {
		t.Fatalf("remove pic missing route id expected 404, got status=%d err=%v", resp4.StatusCode, err)
	}

	resp5, err := app.Test(httptest.NewRequest(http.MethodPost, "/patients//delete", nil))
	if err != nil || resp5.StatusCode != fiber.StatusNotFound {
		t.Fatalf("delete missing route id expected 404, got status=%d err=%v", resp5.StatusCode, err)
	}

	resp6, err := app.Test(httptest.NewRequest(http.MethodGet, "/patients/search?searchTerm=ab", nil))
	if err != nil || resp6.StatusCode != fiber.StatusSeeOther {
		t.Fatalf("index search short term expected 303, got status=%d err=%v", resp6.StatusCode, err)
	}

	resp8, err := app.Test(httptest.NewRequest(http.MethodGet, "/patients//profilepic/", nil))
	if err != nil || resp8.StatusCode != fiber.StatusNotFound {
		t.Fatalf("profile pic missing params expected 404, got status=%d err=%v", resp8.StatusCode, err)
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
		{"short term without htmx", "ab", false, fiber.StatusSeeOther, "at least 3 characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, user := newPatientServiceForTests(t)

			app := fiber.New()
			app.Get("/patients/search", func(c fiber.Ctx) error {
				c.Locals("current_user", user)
				return svc.HandlePatientsIndexSearch(c)
			})

			req := httptest.NewRequest(http.MethodGet, "/patients/search?searchTerm="+tt.term, nil)
			if tt.htmx {
				req.Header.Set("HX-Request", "true")
			}
			resp, err := app.Test(req)
			if err != nil || resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got status=%d err=%v", tt.wantStatus, resp.StatusCode, err)
			}
			if !tt.htmx {
				location, _ := url.QueryUnescape(resp.Header.Get("Location"))
				if !strings.HasPrefix(location, "/patients?toast=") || !strings.Contains(location, tt.wantToast) {
					t.Errorf("redirect = %q, want /patients with the toast", location)
				}
				return
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
			if strings.Contains(string(body), `id="patients_list"`) {
				t.Errorf("search replaced the list")
			}
		})
	}
}
