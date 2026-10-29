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

	appointment, err := svc.AppointmentForShowPage(ctx, 1, "")
	if err != nil {
		t.Fatalf("expected empty-id shortcut without error, got %v", err)
	}
	if appointment.UID != "" {
		t.Fatalf("expected zero-value appointment uid, got %q", appointment.UID)
	}

	appointment, err = svc.AppointmentForShowPage(ctx, 1, "new")
	if err != nil {
		t.Fatalf("expected new-id shortcut without error, got %v", err)
	}
	if appointment.UID != "new" {
		t.Fatalf("expected passthrough appointment uid, got %q", appointment.UID)
	}
}

func TestTakePatientAndClinicOrZero(t *testing.T) {
	svc, user, clinic, patient := newAppointmentServiceForTests(t)
	ctx := t.Context()

	t.Run("blank uids skip the lookup", func(t *testing.T) {
		gotPatient, err := svc.takePatientOrZero(ctx, user.ID, "   ")
		if err != nil || gotPatient.ID != 0 {
			t.Fatalf("blank patient uid = (%d, %v), want zero patient and no error", gotPatient.ID, err)
		}

		gotClinic, err := svc.takeClinicOrZero(ctx, user.ID, "")
		if err != nil || gotClinic.ID != 0 {
			t.Fatalf("blank clinic uid = (%d, %v), want zero clinic and no error", gotClinic.ID, err)
		}
	})

	t.Run("existing uids return the records", func(t *testing.T) {
		gotPatient, err := svc.takePatientOrZero(ctx, user.ID, patient.UID)
		if err != nil || gotPatient.ID != patient.ID {
			t.Fatalf("patient = (%d, %v), want %d", gotPatient.ID, err, patient.ID)
		}

		gotClinic, err := svc.takeClinicOrZero(ctx, user.ID, " "+clinic.UID+" ")
		if err != nil || gotClinic.ID != clinic.ID {
			t.Fatalf("clinic = (%d, %v), want %d", gotClinic.ID, err, clinic.ID)
		}
	})

	t.Run("missing uids return not found", func(t *testing.T) {
		if _, err := svc.takePatientOrZero(ctx, user.ID, "missing"); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("missing patient err = %v, want record not found", err)
		}
		if _, err := svc.takeClinicOrZero(ctx, user.ID, "missing"); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("missing clinic err = %v, want record not found", err)
		}
	})
}

func TestRenderAppointmentNotFoundPage(t *testing.T) {
	svc, _, _, _ := newAppointmentServiceForTests(t)

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
}

func TestRespondWithRedirect(t *testing.T) {
	svc := &service{Server: &server.Server{}}

	t.Run("non-htmx redirects", func(t *testing.T) {
		app := fiber.New()
		app.Get("/appointments", func(c fiber.Ctx) error {
			return svc.respondWithRedirect(c, "/appointments?toast=ok")
		})

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/appointments", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
	})

	t.Run("htmx sets HX-Redirect and status", func(t *testing.T) {
		app := fiber.New()
		app.Get("/appointments", func(c fiber.Ctx) error {
			return svc.respondWithRedirect(c, "/appointments?toast=ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/appointments", nil)
		req.Header.Set("HX-Request", "true")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusNoContent {
			t.Fatalf("expected 204, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("HX-Redirect"); got != "/appointments?toast=ok" {
			t.Fatalf("expected HX-Redirect header, got %q", got)
		}
	})
}
