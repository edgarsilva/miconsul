package tests

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
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

		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204 for unknown appointment update redirect, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("HX-Redirect"); got == "" {
			t.Fatalf("expected HX-Redirect for htmx not found response")
		}
	})

	t.Run("unauthenticated update returns unauthorized for json clients", func(t *testing.T) {
		patient := h.createPatient(u.ID, "Patient Auth")
		clinic := h.createClinic(u.ID, "Clinic Auth")
		appt := h.createAppointment(u.ID, patient.ID, clinic.ID)

		resp, _ := h.doRequest(requestOptions{
			method: http.MethodPost,
			path:   "/appointments/" + appt.UID + "/patch",
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
			path:       "/appointments/" + appt.UID + "/patch",
			authToken:  token,
			htmx:       true,
			contentTyp: "application/json",
			body: url.Values{
				"duration": {"30"},
			},
		})

		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204 for malformed appointment input redirect, got %d", resp.StatusCode)
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

		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204 for invalid appointment create redirect, got %d", resp.StatusCode)
		}
	})

	t.Run("cancel updates status and canceled timestamp", func(t *testing.T) {
		patient := h.createPatient(u.ID, "Patient Cancel")
		clinic := h.createClinic(u.ID, "Clinic Cancel")
		appt := h.createAppointment(u.ID, patient.ID, clinic.ID)

		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodPost,
			path:      "/appointments/" + appt.UID + "/cancel",
			authToken: token,
			htmx:      true,
		})
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204 for appointment cancel redirect, got %d", resp.StatusCode)
		}

		updated, err := gorm.G[models.Appointment](h.db.GormDB()).Where("id = ?", appt.ID).Take(t.Context())
		if err != nil {
			t.Fatalf("load canceled appointment: %v", err)
		}
		if updated.Status != models.AppointmentCanceled {
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
			path:      "/appointments/" + appt.UID + "/complete",
			authToken: token,
			htmx:      true,
			body: url.Values{
				"summary":      {"Procedure completed"},
				"observations": {"Patient recovered well"},
				"conclusions":  {"Follow-up in two weeks"},
				"notes":        {"No complications"},
			},
		})
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204 for appointment complete redirect, got %d", resp.StatusCode)
		}

		updated, err := gorm.G[models.Appointment](h.db.GormDB()).Where("id = ?", appt.ID).Take(t.Context())
		if err != nil {
			t.Fatalf("load completed appointment: %v", err)
		}
		if updated.Status != models.AppointmentDone {
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

		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204 for unknown appointment cancel redirect, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("HX-Redirect"); got == "" {
			t.Fatalf("expected HX-Redirect for htmx cancel not found response")
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

		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204 for unknown appointment complete redirect, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("HX-Redirect"); got == "" {
			t.Fatalf("expected HX-Redirect for htmx complete not found response")
		}
	})

	t.Run("delete unknown appointment redirects", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodDelete,
			path:      "/appointments/apnt_missing",
			authToken: token,
			htmx:      true,
		})

		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204 for htmx unknown appointment delete, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("HX-Redirect"); got != "/appointments" {
			t.Fatalf("HX-Redirect = %q, want /appointments", got)
		}
	})

	t.Run("cross-user update and delete are scoped", func(t *testing.T) {
		owner := h.createUser(models.UserRoleUser)
		ownerPatient := h.createPatient(owner.ID, "Owner Patient")
		ownerClinic := h.createClinic(owner.ID, "Owner Clinic")
		ownerAppt := h.createAppointment(owner.ID, ownerPatient.ID, ownerClinic.ID)

		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodPost,
			path:      "/appointments/" + ownerAppt.UID + "/patch",
			authToken: token,
			htmx:      true,
			body: url.Values{
				"clinicId":  {ownerClinic.UID},
				"patientId": {ownerPatient.UID},
				"duration":  {"30"},
				"bookedAt":  {time.Now().Add(3 * time.Hour).Format("2006-01-02T15:04")},
			},
		})
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204 for cross-user appointment update redirect, got %d", resp.StatusCode)
		}

		resp, _ = h.doRequest(requestOptions{
			method:    http.MethodDelete,
			path:      "/appointments/" + ownerAppt.UID,
			authToken: token,
			htmx:      true,
		})
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204 for htmx cross-user appointment delete, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("HX-Redirect"); got != "/appointments" {
			t.Fatalf("HX-Redirect = %q, want /appointments", got)
		}

		unchanged, err := gorm.G[models.Appointment](h.db.GormDB()).Where("id = ?", ownerAppt.ID).Take(t.Context())
		if err != nil {
			t.Fatalf("load owner appointment after cross-user attempts: %v", err)
		}
		if unchanged.UserID != owner.ID {
			t.Fatalf("expected appointment ownership unchanged, got %d", unchanged.UserID)
		}
	})
}

func TestAppointmentEditPageSelectsItsClinicAndPatient(t *testing.T) {
	tests := []struct {
		name  string
		newer int
	}{
		{"listed but not first", 1},
		{"older than the 10 most recent", 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHarness(t)
			u := h.createUser(models.UserRoleUser)
			clinic := h.createClinic(u.ID, "Booked Clinic")
			patient := h.createPatient(u.ID, "Booked Patient")
			appt := h.createAppointment(u.ID, patient.ID, clinic.ID)

			var newestClinic models.Clinic
			var newestPatient models.Patient
			for i := range tt.newer {
				newestClinic = h.createClinic(u.ID, fmt.Sprintf("Clinic %d", i))
				newestPatient = h.createPatient(u.ID, fmt.Sprintf("Patient %d", i))
			}

			resp, body := h.doRequest(requestOptions{method: http.MethodGet, path: "/appointments/" + appt.UID, authToken: h.authToken(u)})
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200 for the edit page, got %d", resp.StatusCode)
			}

			for _, radio := range []struct {
				name, uid string
				checked   bool
			}{
				{"booked clinic", clinic.UID, true},
				{"booked patient", patient.UID, true},
				{"newest clinic", newestClinic.UID, false},
				{"newest patient", newestPatient.UID, false},
			} {
				tag, ok := radioInput(body, radio.uid)
				if !ok {
					t.Fatalf("expected a radio for the %s", radio.name)
				}
				if got := strings.Contains(tag, " checked"); got != radio.checked {
					t.Errorf("%s radio checked = %v, want %v: %s", radio.name, got, radio.checked, tag)
				}
			}
		})
	}
}

// radioInput returns the <input> tag whose value is uid.
func radioInput(body, uid string) (string, bool) {
	i := strings.Index(body, `value="`+uid+`"`)
	if i < 0 {
		return "", false
	}

	start := strings.LastIndex(body[:i], "<input")
	end := strings.Index(body[i:], ">")
	if start < 0 || end < 0 {
		return "", false
	}

	return body[start : i+end+1], true
}
