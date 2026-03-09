package tests

import (
	"net/http"
	"testing"

	"miconsul/internal/model"
)

func TestAdminHandlers(t *testing.T) {
	h := newTestHarness(t)
	admin := h.createUser(model.UserRoleAdmin)
	regular := h.createUser(model.UserRoleUser)

	t.Run("admin models requires admin role", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{method: http.MethodGet, path: "/admin/models", authToken: h.authToken(regular)})
		if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("expected redirect for non-admin request, got %d", resp.StatusCode)
		}
	})

	t.Run("admin models renders for admin user", func(t *testing.T) {
		resp, body := h.doRequest(requestOptions{method: http.MethodGet, path: "/admin/models", authToken: h.authToken(admin)})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for admin request, got %d", resp.StatusCode)
		}
		if body == "" {
			t.Fatalf("expected non-empty admin models response")
		}
	})
}
