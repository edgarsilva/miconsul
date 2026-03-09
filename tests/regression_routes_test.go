package tests

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"miconsul/internal/model"

	"gorm.io/gorm"
)

func TestProfilePOSTPersistsUpdates(t *testing.T) {
	h := newTestHarness(t)
	u := h.createUser(model.UserRoleUser)
	token := h.authToken(u)

	resp, _ := h.doRequest(requestOptions{
		method:    http.MethodPost,
		path:      "/profile",
		authToken: token,
		body: url.Values{
			"name":  {"Updated Name"},
			"email": {"updated@example.com"},
			"phone": {"555-0199"},
		},
	})

	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected redirect status, got %d", resp.StatusCode)
	}

	updated, err := gorm.G[model.User](h.db.GormDB()).Where("id = ?", u.ID).Take(t.Context())
	if err != nil {
		t.Fatalf("load updated user: %v", err)
	}

	if updated.Name != "Updated Name" || updated.Email != "updated@example.com" || updated.Phone != "555-0199" {
		t.Fatalf("unexpected persisted profile: %+v", updated)
	}
}

func TestPatchAndDeleteRoutesForPatientAndAppointment(t *testing.T) {
	h := newTestHarness(t)
	u := h.createUser(model.UserRoleUser)
	token := h.authToken(u)

	t.Run("patient patch and delete", func(t *testing.T) {
		patient := h.createPatient(u.ID, "Patient One")

		patchResp, _ := h.doRequest(requestOptions{
			method:    http.MethodPatch,
			path:      "/patients/" + patient.ID,
			authToken: token,
			htmx:      true,
			body: url.Values{
				"name":  {"Patient Updated"},
				"phone": {"555-0101"},
				"age":   {"35"},
			},
		})
		if patchResp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for patient patch, got %d", patchResp.StatusCode)
		}

		updated, err := gorm.G[model.Patient](h.db.GormDB()).Where("id = ?", patient.ID).Take(t.Context())
		if err != nil {
			t.Fatalf("load patched patient: %v", err)
		}
		if updated.Name != "Patient Updated" {
			t.Fatalf("expected patient name update, got %q", updated.Name)
		}

		delResp, _ := h.doRequest(requestOptions{
			method:    http.MethodDelete,
			path:      "/patients/" + patient.ID,
			authToken: token,
			htmx:      true,
		})
		if delResp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for patient delete, got %d", delResp.StatusCode)
		}

		_, err = gorm.G[model.Patient](h.db.GormDB()).Where("id = ?", patient.ID).Take(t.Context())
		if err == nil {
			t.Fatalf("expected deleted patient to be missing")
		}
	})

	t.Run("appointment patch and delete", func(t *testing.T) {
		patient := h.createPatient(u.ID, "Patient For Appointment")
		clinic := h.createClinic(u.ID, "Clinic For Appointment")
		appt := h.createAppointment(u.ID, patient.ID, clinic.ID)

		patchResp, _ := h.doRequest(requestOptions{
			method:    http.MethodPost,
			path:      "/appointments/" + appt.ID + "/patch",
			authToken: token,
			htmx:      true,
			body: url.Values{
				"clinicId":  {clinic.ID},
				"patientId": {patient.ID},
				"duration":  {"45"},
				"bookedAt":  {time.Now().Add(3 * time.Hour).Format("2006-01-02T15:04")},
			},
		})
		if patchResp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for appointment patch, got %d", patchResp.StatusCode)
		}

		updated, err := gorm.G[model.Appointment](h.db.GormDB()).Where("id = ?", appt.ID).Take(t.Context())
		if err != nil {
			t.Fatalf("load patched appointment: %v", err)
		}
		if updated.Duration != 45 {
			t.Fatalf("expected appointment duration update, got %d", updated.Duration)
		}

		delResp, _ := h.doRequest(requestOptions{
			method:    http.MethodDelete,
			path:      "/appointments/" + appt.ID,
			authToken: token,
			htmx:      true,
		})
		if delResp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for appointment delete, got %d", delResp.StatusCode)
		}

		_, err = gorm.G[model.Appointment](h.db.GormDB()).Where("id = ?", appt.ID).Take(t.Context())
		if err == nil {
			t.Fatalf("expected deleted appointment to be missing")
		}
	})
}

func TestAPIUsersAuthGating(t *testing.T) {
	h := newTestHarness(t)
	admin := h.createUser(model.UserRoleAdmin)
	regular := h.createUser(model.UserRoleUser)

	t.Run("unauthenticated denied", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{method: http.MethodGet, path: "/api/users", accept: "application/json"})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("non admin forbidden", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{method: http.MethodGet, path: "/api/users", accept: "application/json", authToken: h.authToken(regular)})
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.StatusCode)
		}
	})

	t.Run("admin allowed", func(t *testing.T) {
		resp, body := h.doRequest(requestOptions{method: http.MethodGet, path: "/api/users", accept: "application/json", authToken: h.authToken(admin)})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if body == "" {
			t.Fatalf("expected JSON body from /api/users")
		}
	})
}

func TestAppointmentsSearchClinicsRouteBehavior(t *testing.T) {
	h := newTestHarness(t)
	u := h.createUser(model.UserRoleUser)
	token := h.authToken(u)

	h.createClinic(u.ID, "Bright Dental")
	h.createClinic(u.ID, "Calm Therapy")

	resp, body := h.doRequest(requestOptions{
		method:    http.MethodPost,
		path:      "/appointments/search/clinics",
		authToken: token,
		htmx:      true,
		body: url.Values{
			"searchTerm": {""},
		},
	})

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if body == "" {
		t.Fatalf("expected non-empty html fragment")
	}
	if !strings.Contains(body, "Bright Dental") {
		t.Fatalf("expected search results to include first clinic")
	}
	if !strings.Contains(body, "Calm Therapy") {
		t.Fatalf("expected search results to include second clinic")
	}
}
