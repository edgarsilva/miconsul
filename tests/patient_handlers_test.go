package tests

import (
	"net/http"
	"net/url"
	"testing"

	"miconsul/internal/model"

	"gorm.io/gorm"
)

func TestPatientHandlers(t *testing.T) {
	h := newTestHarness(t)
	u := h.createUser(model.UserRoleUser)
	token := h.authToken(u)

	t.Run("update patient htmx returns success redirect header", func(t *testing.T) {
		patient := h.createPatient(u.ID, "Patient One")

		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodPost,
			path:      "/patients/" + patient.ID + "/patch",
			authToken: token,
			htmx:      true,
			body: url.Values{
				"name":  {"Patient Updated"},
				"phone": {"555-0142"},
				"age":   {"31"},
			},
		})

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for patient update, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("HX-Location"); got == "" {
			t.Fatalf("expected HX-Location to be set for htmx patient update")
		}

		updated, err := gorm.G[model.Patient](h.db.GormDB()).Where("id = ?", patient.ID).Take(t.Context())
		if err != nil {
			t.Fatalf("load updated patient: %v", err)
		}
		if updated.Name != "Patient Updated" {
			t.Fatalf("expected patient name update, got %q", updated.Name)
		}
	})

	t.Run("update returns not found for unknown patient", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodPost,
			path:      "/patients/ptnt_missing/patch",
			authToken: token,
			htmx:      true,
			body: url.Values{
				"name":  {"Missing"},
				"phone": {"555-0100"},
				"age":   {"21"},
			},
		})

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 for unknown patient update, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("HX-Location"); got == "" {
			t.Fatalf("expected HX-Location redirect for htmx patient not found")
		}
	})
}
