package tests

import (
	"net/http"
	"strings"
	"testing"

	"miconsul/internal/models"
)

func TestAdminHandlers(t *testing.T) {
	h := newTestHarness(t)
	admin := h.createUser(models.UserRoleAdmin)

	t.Run("components page is for admins only", func(t *testing.T) {
		resp, body := h.doRequest(requestOptions{path: "/admin/components", authToken: h.authToken(admin)})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for admins, got %d", resp.StatusCode)
		}
		for _, name := range []string{"CmpBackBtn", "CmpStatusBadge", "CmpCalendarIcon", "Patient row (design reference)"} {
			if !strings.Contains(body, name) {
				t.Errorf("expected the components page to show %s", name)
			}
		}

		user := h.createUser(models.UserRoleUser)
		resp, _ = h.doRequest(requestOptions{path: "/admin/components", authToken: h.authToken(user)})
		if resp.StatusCode == http.StatusOK {
			t.Fatalf("expected non-admins to be turned away, got 200")
		}
	})

	t.Run("admin jobs ui is disabled by default", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{method: http.MethodGet, path: "/admin/jobs", authToken: h.authToken(admin)})
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 when jobs ui disabled, got %d", resp.StatusCode)
		}
	})
}
