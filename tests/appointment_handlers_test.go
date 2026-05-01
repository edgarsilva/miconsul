package tests

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"miconsul/internal/models"

	"gorm.io/gorm"
)

func TestAppointmentHandlers(t *testing.T) {
	h := newTestHarness(t)
	u := h.createUser(models.UserRoleUser)
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

	t.Run("unauthenticated update returns unauthorized for json clients", func(t *testing.T) {
		patient := h.createPatient(u.ID, "Patient Auth")
		clinic := h.createClinic(u.ID, "Clinic Auth")
		appt := h.createAppointment(u.ID, patient.ID, clinic.ID)

		resp, _ := h.doRequest(requestOptions{
			method: http.MethodPost,
			path:   "/appointments/" + appt.ID + "/patch",
			accept: "application/json",
			body: url.Values{
				"duration": {"30"},
				"bookedAt": {time.Now().Add(2 * time.Hour).Format("2006-01-02T15:04")},
			},
		})

		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 for unauthenticated appointment update, got %d", resp.StatusCode)
		}
	})

	t.Run("update returns bad request for malformed input", func(t *testing.T) {
		patient := h.createPatient(u.ID, "Patient Bad Request")
		clinic := h.createClinic(u.ID, "Clinic Bad Request")
		appt := h.createAppointment(u.ID, patient.ID, clinic.ID)

		resp, _ := h.doRequest(requestOptions{
			method:     http.MethodPost,
			path:       "/appointments/" + appt.ID + "/patch",
			authToken:  token,
			htmx:       true,
			contentTyp: "application/json",
			body: url.Values{
				"duration": {"30"},
			},
		})

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for malformed appointment input, got %d", resp.StatusCode)
		}
	})

	t.Run("create returns unprocessable for missing required relationships", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodPost,
			path:      "/appointments/",
			authToken: token,
			htmx:      true,
			body: url.Values{
				"clinicId":  {""},
				"patientId": {""},
				"duration":  {"30"},
			},
		})

		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422 for invalid appointment create payload, got %d", resp.StatusCode)
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

		updated, err := gorm.G[models.Appointment](h.db.GormDB()).Where("id = ?", appt.ID).Take(t.Context())
		if err != nil {
			t.Fatalf("load canceled appointment: %v", err)
		}
		if updated.Status != models.ApntStatusCanceled {
			t.Fatalf("expected canceled status, got %q", updated.Status)
		}
		if updated.CanceledAt.IsZero() {
			t.Fatalf("expected canceled timestamp to be set")
		}
	})

	t.Run("complete updates status and completion notes", func(t *testing.T) {
		patient := h.createPatient(u.ID, "Patient Complete")
		clinic := h.createClinic(u.ID, "Clinic Complete")
		appt := h.createAppointment(u.ID, patient.ID, clinic.ID)

		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodPost,
			path:      "/appointments/" + appt.ID + "/complete",
			authToken: token,
			htmx:      true,
			body: url.Values{
				"summary":      {"Procedure completed"},
				"observations": {"Patient recovered well"},
				"conclusions":  {"Follow-up in two weeks"},
				"notes":        {"No complications"},
			},
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for appointment complete, got %d", resp.StatusCode)
		}

		updated, err := gorm.G[models.Appointment](h.db.GormDB()).Where("id = ?", appt.ID).Take(t.Context())
		if err != nil {
			t.Fatalf("load completed appointment: %v", err)
		}
		if updated.Status != models.ApntStatusDone {
			t.Fatalf("expected done status, got %q", updated.Status)
		}
		if updated.Summary != "Procedure completed" || updated.Notes != "No complications" {
			t.Fatalf("expected completion notes to persist, got summary=%q notes=%q", updated.Summary, updated.Notes)
		}
	})

	t.Run("cancel unknown appointment returns not found", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodPost,
			path:      "/appointments/apnt_missing/cancel",
			authToken: token,
			htmx:      true,
		})

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 for unknown appointment cancel, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("HX-Location"); got == "" {
			t.Fatalf("expected HX-Location redirect for htmx cancel not found response")
		}
	})

	t.Run("complete unknown appointment returns not found", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodPost,
			path:      "/appointments/apnt_missing/complete",
			authToken: token,
			htmx:      true,
			body: url.Values{
				"summary": {"complete missing"},
			},
		})

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 for unknown appointment complete, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("HX-Location"); got == "" {
			t.Fatalf("expected HX-Location redirect for htmx complete not found response")
		}
	})

	t.Run("delete unknown appointment redirects", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodDelete,
			path:      "/appointments/apnt_missing",
			authToken: token,
			htmx:      true,
		})

		if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("expected redirect for unknown appointment delete, got %d", resp.StatusCode)
		}
	})

	t.Run("cross-user update and delete are scoped", func(t *testing.T) {
		owner := h.createUser(models.UserRoleUser)
		ownerPatient := h.createPatient(owner.ID, "Owner Patient")
		ownerClinic := h.createClinic(owner.ID, "Owner Clinic")
		ownerAppt := h.createAppointment(owner.ID, ownerPatient.ID, ownerClinic.ID)

		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodPost,
			path:      "/appointments/" + ownerAppt.ID + "/patch",
			authToken: token,
			htmx:      true,
			body: url.Values{
				"clinicId":  {ownerClinic.ID},
				"patientId": {ownerPatient.ID},
				"duration":  {"30"},
				"bookedAt":  {time.Now().Add(3 * time.Hour).Format("2006-01-02T15:04")},
			},
		})
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 for cross-user appointment update, got %d", resp.StatusCode)
		}

		resp, _ = h.doRequest(requestOptions{
			method:    http.MethodDelete,
			path:      "/appointments/" + ownerAppt.ID,
			authToken: token,
			htmx:      true,
		})
		if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("expected redirect for cross-user appointment delete, got %d", resp.StatusCode)
		}

		unchanged, err := gorm.G[models.Appointment](h.db.GormDB()).Where("id = ?", ownerAppt.ID).Take(t.Context())
		if err != nil {
			t.Fatalf("load owner appointment after cross-user attempts: %v", err)
		}
		if unchanged.UserID != owner.ID {
			t.Fatalf("expected appointment ownership unchanged, got %q", unchanged.UserID)
		}
	})
}
