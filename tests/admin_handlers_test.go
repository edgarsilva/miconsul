package tests

import (
	"net/http"
	"testing"

	"miconsul/internal/models"
)

func TestAdminHandlers(t *testing.T) {
	h := newTestHarness(t)
	admin := h.createUser(models.UserRoleAdmin)

	t.Run("admin jobs ui is disabled by default", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{method: http.MethodGet, path: "/admin/jobs", authToken: h.authToken(admin)})
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 when jobs ui disabled, got %d", resp.StatusCode)
		}
	})
}
