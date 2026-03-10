package appointment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"miconsul/internal/server"

	"github.com/gofiber/fiber/v3"
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
