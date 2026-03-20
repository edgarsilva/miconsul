package tests

import (
	"net/http"
	"testing"

	"miconsul/internal/models"
)

func TestDashboardHandlers(t *testing.T) {
	h := newTestHarness(t)
	u := h.createUser(model.UserRoleUser)
	token := h.authToken(u)

	t.Run("dashboard requires authentication", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{method: http.MethodGet, path: "/dashboard"})
		if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("expected redirect for unauthenticated dashboard request, got %d", resp.StatusCode)
		}
	})

	t.Run("dashboard renders appointments for authenticated user", func(t *testing.T) {
		patient := h.createPatient(u.ID, "Dashboard Patient")
		clinic := h.createClinic(u.ID, "Dashboard Clinic")
		h.createAppointment(u.ID, patient.ID, clinic.ID)

		resp, body := h.doRequest(requestOptions{method: http.MethodGet, path: "/dashboard?timeframe=day", authToken: token})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for authenticated dashboard request, got %d", resp.StatusCode)
		}
		if body == "" {
			t.Fatalf("expected non-empty dashboard response body")
		}
	})
}
