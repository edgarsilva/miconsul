package tests

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"miconsul/internal/models"

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

	t.Run("update with unchanged values remains success", func(t *testing.T) {
		patient := h.createPatient(u.ID, "Patient Stable")

		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodPost,
			path:      "/patients/" + patient.ID + "/patch",
			authToken: token,
			htmx:      true,
			body: url.Values{
				"name":  {"Patient Stable"},
				"phone": {"555-0100"},
				"age":   {"30"},
			},
		})

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for unchanged patient update, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("HX-Location"); got == "" {
			t.Fatalf("expected HX-Location for unchanged patient update")
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

	t.Run("update returns bad request for malformed input", func(t *testing.T) {
		patient := h.createPatient(u.ID, "Patient Malformed")

		resp, _ := h.doRequest(requestOptions{
			method:     http.MethodPost,
			path:       "/patients/" + patient.ID + "/patch",
			authToken:  token,
			htmx:       true,
			contentTyp: "application/json",
			body: url.Values{
				"name": {"Malformed"},
			},
		})

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for malformed patient input, got %d", resp.StatusCode)
		}
	})

	t.Run("update returns unprocessable for invalid boundaries", func(t *testing.T) {
		patient := h.createPatient(u.ID, "Patient Invalid")

		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodPost,
			path:      "/patients/" + patient.ID + "/patch",
			authToken: token,
			htmx:      true,
			body: url.Values{
				"name":  {strings.Repeat("a", 121)},
				"phone": {"555-0100"},
				"age":   {"30"},
			},
		})

		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422 for invalid patient boundaries, got %d", resp.StatusCode)
		}
	})

	t.Run("cross-user update and delete are scoped", func(t *testing.T) {
		owner := h.createUser(model.UserRoleUser)
		ownerPatient := h.createPatient(owner.ID, "Owner Patient")

		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodPost,
			path:      "/patients/" + ownerPatient.ID + "/patch",
			authToken: token,
			htmx:      true,
			body: url.Values{
				"name":  {"Attempted Hijack"},
				"phone": {"555-0999"},
				"age":   {"40"},
			},
		})
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 for cross-user patient update, got %d", resp.StatusCode)
		}

		resp, _ = h.doRequest(requestOptions{
			method:    http.MethodDelete,
			path:      "/patients/" + ownerPatient.ID,
			authToken: token,
			htmx:      true,
		})
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 for cross-user patient delete, got %d", resp.StatusCode)
		}

		unchanged, err := gorm.G[model.Patient](h.db.GormDB()).Where("id = ?", ownerPatient.ID).Take(t.Context())
		if err != nil {
			t.Fatalf("load owner patient after cross-user attempts: %v", err)
		}
		if unchanged.Name != "Owner Patient" {
			t.Fatalf("expected owner patient name unchanged, got %q", unchanged.Name)
		}
	})
}
