package appointment

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"miconsul/internal/server"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func TestAppointmentForShowPage(t *testing.T) {
	svc := &service{}
	ctx := context.Background()

	appointment, err := svc.AppointmentForShowPage(ctx, "usr_1", "")
	if err != nil {
		t.Fatalf("expected empty-id shortcut without error, got %v", err)
	}
	if appointment.ID != "" {
		t.Fatalf("expected zero-value appointment id, got %q", appointment.ID)
	}

	appointment, err = svc.AppointmentForShowPage(ctx, "usr_1", "new")
	if err != nil {
		t.Fatalf("expected new-id shortcut without error, got %v", err)
	}
	if appointment.ID != "new" {
		t.Fatalf("expected passthrough appointment id, got %q", appointment.ID)
	}
}

func TestPatientAndClinicSelectionHelpers(t *testing.T) {
	svc := &service{}
	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		patient, err := svc.selectedPatientFromQuery(c, "usr_1", "   ")
		if err != nil {
			t.Fatalf("expected blank patient query to skip lookup, got %v", err)
		}
		if patient.ID != "" {
			t.Fatalf("expected zero patient when query is blank, got %q", patient.ID)
		}

		clinic, err := svc.selectedClinicFromQuery(c, "usr_1", "")
		if err != nil {
			t.Fatalf("expected blank clinic query to skip lookup, got %v", err)
		}
		if clinic.ID != "" {
			t.Fatalf("expected zero clinic when query is blank, got %q", clinic.ID)
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected status 204, got %d", resp.StatusCode)
	}
}

func TestSelectionHelpersBranchesAndNotFoundPage(t *testing.T) {
	svc, user, clinic, patient := newAppointmentServiceForTests(t)

	t.Run("selected helpers return entities when ids exist", func(t *testing.T) {
		app := fiber.New()
		app.Get("/", func(c fiber.Ctx) error {
			gotPatient, err := svc.selectedPatientFromQuery(c, user.ID, patient.ID)
			if err != nil {
				t.Fatalf("expected selected patient success, got %v", err)
			}
			if gotPatient.ID != patient.ID {
				t.Fatalf("expected patient id %q, got %q", patient.ID, gotPatient.ID)
			}

			gotClinic, err := svc.selectedClinicFromQuery(c, user.ID, clinic.ID)
			if err != nil {
				t.Fatalf("expected selected clinic success, got %v", err)
			}
			if gotClinic.ID != clinic.ID {
				t.Fatalf("expected clinic id %q, got %q", clinic.ID, gotClinic.ID)
			}

			return c.SendStatus(fiber.StatusNoContent)
		})

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusNoContent {
			t.Fatalf("expected 204, got %d", resp.StatusCode)
		}
	})

	t.Run("selected helpers bubble not-found errors for missing ids", func(t *testing.T) {
		app := fiber.New()
		app.Get("/", func(c fiber.Ctx) error {
			_, err := svc.selectedPatientFromQuery(c, user.ID, "missing")
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				t.Fatalf("expected patient record not found, got %v", err)
			}

			_, err = svc.selectedClinicFromQuery(c, user.ID, "missing")
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				t.Fatalf("expected clinic record not found, got %v", err)
			}

			return c.SendStatus(fiber.StatusNoContent)
		})

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusNoContent {
			t.Fatalf("expected 204, got %d", resp.StatusCode)
		}
	})

	t.Run("renders appointment not found page", func(t *testing.T) {
		app := fiber.New()
		app.Get("/", func(c fiber.Ctx) error {
			return svc.renderAppointmentNotFoundPage(c, "Missing appointment", "warning")
		})

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
	})
}

func TestRespondWithRedirect(t *testing.T) {
	svc := &service{Server: &server.Server{}}

	t.Run("non-htmx redirects", func(t *testing.T) {
		app := fiber.New()
		app.Get("/appointments", func(c fiber.Ctx) error {
			return svc.respondWithRedirect(c, "/appointments?toast=ok", fiber.StatusBadRequest)
		})

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/appointments", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
	})

	t.Run("htmx sets HX-Location and status", func(t *testing.T) {
		app := fiber.New()
		app.Get("/appointments", func(c fiber.Ctx) error {
			return svc.respondWithRedirect(c, "/appointments?toast=ok", fiber.StatusBadRequest)
		})

		req := httptest.NewRequest(http.MethodGet, "/appointments", nil)
		req.Header.Set("HX-Request", "true")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("HX-Location"); got != "/appointments?toast=ok" {
			t.Fatalf("expected HX-Location header, got %q", got)
		}
	})
}
