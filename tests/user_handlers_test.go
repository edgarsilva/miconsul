package tests

import (
	"net/http"
	"net/url"
	"testing"

	"miconsul/internal/models"

	"gorm.io/gorm"
)

func TestUserHandlers(t *testing.T) {
	h := newTestHarness(t)
	admin := h.createUser(model.UserRoleAdmin)
	regular := h.createUser(model.UserRoleUser)

	t.Run("profile update persists and redirects", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodPost,
			path:      "/profile",
			authToken: h.authToken(regular),
			body: url.Values{
				"name":  {"Updated Name"},
				"email": {"updated@example.com"},
				"phone": {"555-0199"},
			},
		})

		if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("expected redirect status, got %d", resp.StatusCode)
		}

		updated, err := gorm.G[model.User](h.db.GormDB()).Where("id = ?", regular.ID).Take(t.Context())
		if err != nil {
			t.Fatalf("load updated user: %v", err)
		}
		if updated.Name != "Updated Name" || updated.Email != "updated@example.com" || updated.Phone != "555-0199" {
			t.Fatalf("unexpected persisted profile: %+v", updated)
		}
	})

	t.Run("api users auth gating", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{method: http.MethodGet, path: "/api/users", accept: "application/json"})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 for unauthenticated request, got %d", resp.StatusCode)
		}

		resp, _ = h.doRequest(requestOptions{method: http.MethodGet, path: "/api/users", accept: "application/json", authToken: h.authToken(regular)})
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 for non-admin request, got %d", resp.StatusCode)
		}

		resp, body := h.doRequest(requestOptions{method: http.MethodGet, path: "/api/users", accept: "application/json", authToken: h.authToken(admin)})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for admin request, got %d", resp.StatusCode)
		}
		if body == "" {
			t.Fatalf("expected non-empty JSON body for admin /api/users")
		}
	})
}
