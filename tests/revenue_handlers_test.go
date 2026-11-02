package tests

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"miconsul/internal/models"
)

func TestRevenueHandlers(t *testing.T) {
	h := newTestHarness(t)
	u := h.createUser(models.UserRoleUser)
	token := h.authToken(u)

	t.Run("page shows the last 12 months and the selected month by clinic", func(t *testing.T) {
		clinic := h.createClinic(u.ID, "Revenue Clinic")
		patient := h.createPatient(u.ID, "Revenue Patient")
		apnt := h.createAppointment(u.ID, patient.ID, clinic.ID)
		err := h.db.GormDB().Model(&models.Appointment{}).Where("id = ?", apnt.ID).
			Updates(map[string]any{"status": models.AppointmentDone, "price": 2450000, "booked_at": time.Now().UTC().Add(-time.Minute)}).Error
		if err != nil {
			t.Fatalf("mark appointment done: %v", err)
		}

		resp, body := h.doRequest(requestOptions{path: "/revenue", authToken: token})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if got := strings.Count(body, `href="/revenue?month=`); got != 12 {
			t.Errorf("expected 12 month links, got %d", got)
		}
		for _, want := range []string{"Revenue Clinic", "$24,500", `href="/revenue"`} {
			if !strings.Contains(body, want) {
				t.Errorf("expected the page to show %q", want)
			}
		}
	})

	t.Run("signed out visitors are turned away", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{path: "/revenue"})
		if resp.StatusCode == http.StatusOK {
			t.Fatalf("expected signed out visitors to be turned away, got 200")
		}
	})
}
