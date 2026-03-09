package tests

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"miconsul/internal/model"

	"gorm.io/gorm"
)

func TestAppointmentHandlers(t *testing.T) {
	h := newTestHarness(t)
	u := h.createUser(model.UserRoleUser)
	token := h.authToken(u)

	t.Run("update returns not found for unknown appointment", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodPost,
			path:      "/appointments/apnt_missing/patch",
			authToken: token,
			htmx:      true,
			body: url.Values{
				"duration": {"30"},
				"bookedAt": {time.Now().Add(2 * time.Hour).Format("2006-01-02T15:04")},
			},
		})

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 for unknown appointment update, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("HX-Location"); got == "" {
			t.Fatalf("expected HX-Location redirect for htmx not found response")
		}
	})

	t.Run("cancel updates status and canceled timestamp", func(t *testing.T) {
		patient := h.createPatient(u.ID, "Patient Cancel")
		clinic := h.createClinic(u.ID, "Clinic Cancel")
		appt := h.createAppointment(u.ID, patient.ID, clinic.ID)

		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodPost,
			path:      "/appointments/" + appt.ID + "/cancel",
			authToken: token,
			htmx:      true,
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for appointment cancel, got %d", resp.StatusCode)
		}

		updated, err := gorm.G[model.Appointment](h.db.GormDB()).Where("id = ?", appt.ID).Take(t.Context())
		if err != nil {
			t.Fatalf("load canceled appointment: %v", err)
		}
		if updated.Status != model.ApntStatusCanceled {
			t.Fatalf("expected canceled status, got %q", updated.Status)
		}
		if updated.CanceledAt.IsZero() {
			t.Fatalf("expected canceled timestamp to be set")
		}
	})
}
