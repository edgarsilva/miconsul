package appointment

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	view "miconsul/internal/views"
)

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
		return svc.renderAppointmentNotFoundPage(c, "Missing appointment", view.LevelWarning)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
