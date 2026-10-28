package tests

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"miconsul/internal/models"

	"gorm.io/gorm"
)

func TestDashboardHandlers(t *testing.T) {
	h := newTestHarness(t)
	u := h.createUser(models.UserRoleUser)
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

func TestDashboardShowsLoadErrors(t *testing.T) {
	h := newTestHarness(t)
	u := h.createUser(models.UserRoleUser)
	err := h.db.GormDB().Callback().Query().Before("gorm:query").Register("test_force_appointments_query_error", func(db *gorm.DB) {
		if db.Statement != nil && db.Statement.Table == "appointments" {
			db.AddError(errors.New("forced appointments query error"))
		}
	})
	if err != nil {
		t.Fatalf("register query callback: %v", err)
	}

	resp, body := h.doRequest(requestOptions{method: http.MethodGet, path: "/dashboard?timeframe=day", authToken: h.authToken(u)})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected the dashboard to render, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, "Failed to load part of the dashboard") {
		t.Fatalf("expected an error toast on the dashboard")
	}
}
