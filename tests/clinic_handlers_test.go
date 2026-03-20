package tests

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"miconsul/internal/models"

	"gorm.io/gorm"
)

func TestClinicHandlers(t *testing.T) {
	h := newTestHarness(t)
	u := h.createUser(models.UserRoleUser)
	token := h.authToken(u)

	t.Run("update missing id returns bad request", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{method: http.MethodPost, path: "/clinics//patch", authToken: token, htmx: true})
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 for unmatched missing-id route, got %d", resp.StatusCode)
		}
	})

	t.Run("update unknown id returns not found", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodPost,
			path:      "/clinics/clnc_missing/patch",
			authToken: token,
			htmx:      true,
			body: url.Values{
				"name": {"Missing Clinic"},
			},
		})

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 for unknown clinic update, got %d", resp.StatusCode)
		}
	})

	t.Run("unauthenticated update returns unauthorized for json clients", func(t *testing.T) {
		clinic := h.createClinic(u.ID, "Clinic Auth")

		resp, _ := h.doRequest(requestOptions{
			method: http.MethodPost,
			path:   "/clinics/" + clinic.ID + "/patch",
			accept: "application/json",
			body: url.Values{
				"name": {"Blocked"},
			},
		})

		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 for unauthenticated clinic update, got %d", resp.StatusCode)
		}
	})

	t.Run("update htmx persists and sets push url", func(t *testing.T) {
		clinic := h.createClinic(u.ID, "Clinic One")

		resp, body := h.doRequest(requestOptions{
			method:    http.MethodPost,
			path:      "/clinics/" + clinic.ID + "/patch",
			authToken: token,
			htmx:      true,
			body: url.Values{
				"name":  {"Clinic Updated"},
				"email": {"clinic@example.com"},
				"phone": {"555-0200"},
			},
		})

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for clinic update, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("HX-Push-Url"); got == "" {
			t.Fatalf("expected HX-Push-Url to be set after htmx clinic update")
		}
		if !strings.Contains(body, "Clinic Updated") {
			t.Fatalf("expected updated clinic name in response body")
		}

		updated, err := gorm.G[models.Clinic](h.db.GormDB()).Where("id = ?", clinic.ID).Take(t.Context())
		if err != nil {
			t.Fatalf("load updated clinic: %v", err)
		}
		if updated.Name != "Clinic Updated" {
			t.Fatalf("expected clinic name update, got %q", updated.Name)
		}
	})

	t.Run("update with unchanged values remains success", func(t *testing.T) {
		clinic := h.createClinic(u.ID, "Clinic Stable")

		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodPost,
			path:      "/clinics/" + clinic.ID + "/patch",
			authToken: token,
			htmx:      true,
			body: url.Values{
				"name":  {"Clinic Stable"},
				"email": {""},
				"phone": {""},
			},
		})

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for unchanged clinic update, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("HX-Push-Url"); got == "" {
			t.Fatalf("expected HX-Push-Url for unchanged clinic update")
		}
	})

	t.Run("update returns unprocessable for invalid boundaries", func(t *testing.T) {
		clinic := h.createClinic(u.ID, "Clinic Invalid")

		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodPost,
			path:      "/clinics/" + clinic.ID + "/patch",
			authToken: token,
			htmx:      true,
			body: url.Values{
				"name": {strings.Repeat("a", 121)},
			},
		})

		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422 for invalid clinic boundaries, got %d", resp.StatusCode)
		}
	})

	t.Run("delete missing id route is method not allowed", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{method: http.MethodDelete, path: "/clinics/", authToken: token, htmx: true})
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405 for missing-id delete route, got %d", resp.StatusCode)
		}
	})

	t.Run("delete unknown id returns not found", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodDelete,
			path:      "/clinics/clnc_missing",
			authToken: token,
			htmx:      true,
		})

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 for unknown clinic delete, got %d", resp.StatusCode)
		}
	})

	t.Run("delete htmx removes row and sets location", func(t *testing.T) {
		clinic := h.createClinic(u.ID, "Clinic Delete")

		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodDelete,
			path:      "/clinics/" + clinic.ID,
			authToken: token,
			htmx:      true,
		})

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for clinic delete, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("HX-Location"); got != "/clinics" {
			t.Fatalf("expected HX-Location /clinics, got %q", got)
		}

		_, err := gorm.G[models.Clinic](h.db.GormDB()).Where("id = ?", clinic.ID).Take(t.Context())
		if err == nil {
			t.Fatalf("expected deleted clinic to be missing")
		}
	})

	t.Run("search term shorter than 3 returns bad request", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodGet,
			path:      "/clinics/search?searchTerm=ab",
			authToken: token,
			htmx:      true,
		})

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for short clinic search term, got %d", resp.StatusCode)
		}
	})

	t.Run("search empty term returns results fragment", func(t *testing.T) {
		h.createClinic(u.ID, "Bright Dental")
		h.createClinic(u.ID, "Calm Therapy")

		resp, body := h.doRequest(requestOptions{
			method:    http.MethodGet,
			path:      "/clinics/search?searchTerm=",
			authToken: token,
			htmx:      true,
		})

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for empty clinic search, got %d", resp.StatusCode)
		}
		if !strings.Contains(body, "Bright Dental") {
			t.Fatalf("expected search results to include clinic")
		}
	})

	t.Run("cross-user update and delete are scoped", func(t *testing.T) {
		owner := h.createUser(models.UserRoleUser)
		ownerClinic := h.createClinic(owner.ID, "Owner Clinic")

		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodPost,
			path:      "/clinics/" + ownerClinic.ID + "/patch",
			authToken: token,
			htmx:      true,
			body: url.Values{
				"name": {"Hijacked Clinic"},
			},
		})
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 for cross-user clinic update, got %d", resp.StatusCode)
		}

		resp, _ = h.doRequest(requestOptions{
			method:    http.MethodDelete,
			path:      "/clinics/" + ownerClinic.ID,
			authToken: token,
			htmx:      true,
		})
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 for cross-user clinic delete, got %d", resp.StatusCode)
		}

		unchanged, err := gorm.G[models.Clinic](h.db.GormDB()).Where("id = ?", ownerClinic.ID).Take(t.Context())
		if err != nil {
			t.Fatalf("load owner clinic after cross-user attempts: %v", err)
		}
		if unchanged.Name != "Owner Clinic" {
			t.Fatalf("expected owner clinic name unchanged, got %q", unchanged.Name)
		}
	})
}
