package appointment

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
)

func TestAppointmentHandlersFlows(t *testing.T) {
	svc, user, clinic, patient := newAppointmentServiceForTests(t)

	apnt := models.Appointment{
		UserID:    user.ID,
		ClinicID:  clinic.ID,
		PatientID: patient.ID,
		BookedAt:  time.Now().Add(time.Hour),
		Token:     "tok_handlers",
	}
	if err := svc.CreateAppointment(t.Context(), &apnt); err != nil {
		t.Fatalf("seed appointment: %v", err)
	}

	t.Run("index and show pages render", func(t *testing.T) {
		app := fiber.New()
		app.Get("/appointments", func(c fiber.Ctx) error {
			c.Locals("current_user", user)
			return svc.HandleIndexPage(c)
		})
		app.Get("/appointments/:id", func(c fiber.Ctx) error {
			c.Locals("current_user", user)
			return svc.HandleShowPage(c)
		})

		resp1, err := app.Test(httptest.NewRequest(http.MethodGet, "/appointments", nil))
		if err != nil || resp1.StatusCode != fiber.StatusOK {
			t.Fatalf("index page expected 200, got status=%d err=%v", resp1.StatusCode, err)
		}

		resp2, err := app.Test(httptest.NewRequest(http.MethodGet, "/appointments/"+apnt.ID, nil))
		if err != nil || resp2.StatusCode != fiber.StatusOK {
			t.Fatalf("show page expected 200, got status=%d err=%v", resp2.StatusCode, err)
		}
	})

	t.Run("start page and create/update/delete endpoints", func(t *testing.T) {
		app := fiber.New()
		app.Get("/appointments/:id/start", func(c fiber.Ctx) error {
			c.Locals("current_user", user)
			return svc.HandleStartPage(c)
		})
		app.Post("/appointments", func(c fiber.Ctx) error {
			c.Locals("current_user", user)
			return svc.HandleCreate(c)
		})
		app.Post("/appointments/:id/patch", func(c fiber.Ctx) error {
			c.Locals("current_user", user)
			return svc.HandleUpdate(c)
		})
		app.Post("/appointments/:id/complete", func(c fiber.Ctx) error {
			c.Locals("current_user", user)
			return svc.HandleComplete(c)
		})
		app.Post("/appointments/:id/cancel", func(c fiber.Ctx) error {
			c.Locals("current_user", user)
			return svc.HandleCancel(c)
		})
		app.Post("/appointments/:id/delete", func(c fiber.Ctx) error {
			c.Locals("current_user", user)
			return svc.HandleDelete(c)
		})
		app.Get("/appointments/new/pricefrg/:id", func(c fiber.Ctx) error {
			c.Locals("current_user", user)
			return svc.HandlePriceFrg(c)
		})
		app.Post("/appointments/search/clinics", func(c fiber.Ctx) error {
			c.Locals("current_user", user)
			return svc.HandleSearchClinics(c)
		})

		resp1, err := app.Test(httptest.NewRequest(http.MethodGet, "/appointments/"+apnt.ID+"/start", nil))
		if err != nil || resp1.StatusCode != fiber.StatusOK {
			t.Fatalf("start page expected 200, got status=%d err=%v", resp1.StatusCode, err)
		}

		form := url.Values{
			"bookedAt":  {time.Now().Format("2006-01-02T15:04")},
			"price":     {"100.0"},
			"clinicId":  {clinic.ID},
			"patientId": {patient.ID},
			"duration":  {"30"},
		}
		req2 := httptest.NewRequest(http.MethodPost, "/appointments", strings.NewReader(form.Encode()))
		req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp2, err := app.Test(req2)
		if err != nil || resp2.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("create expected 303, got status=%d err=%v", resp2.StatusCode, err)
		}

		updForm := url.Values{
			"bookedAt":  {time.Now().Add(2 * time.Hour).Format("2006-01-02T15:04")},
			"price":     {"120.0"},
			"clinicId":  {clinic.ID},
			"patientId": {patient.ID},
			"duration":  {"45"},
		}
		req3 := httptest.NewRequest(http.MethodPost, "/appointments/"+apnt.ID+"/patch", strings.NewReader(updForm.Encode()))
		req3.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp3, err := app.Test(req3)
		if err != nil || resp3.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("update expected 303, got status=%d err=%v", resp3.StatusCode, err)
		}

		completeForm := url.Values{
			"observations": {"all good"},
			"conclusions":  {"done"},
			"summary":      {"ok"},
			"notes":        {"notes"},
		}
		req4 := httptest.NewRequest(http.MethodPost, "/appointments/"+apnt.ID+"/complete", strings.NewReader(completeForm.Encode()))
		req4.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp4, err := app.Test(req4)
		if err != nil || resp4.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("complete expected 303, got status=%d err=%v", resp4.StatusCode, err)
		}

		resp5, err := app.Test(httptest.NewRequest(http.MethodPost, "/appointments/"+apnt.ID+"/cancel", nil))
		if err != nil || resp5.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("cancel expected 303, got status=%d err=%v", resp5.StatusCode, err)
		}

		resp6, err := app.Test(httptest.NewRequest(http.MethodGet, "/appointments/new/pricefrg/"+clinic.ID, nil))
		if err != nil || resp6.StatusCode != fiber.StatusOK {
			t.Fatalf("price fragment expected 200, got status=%d err=%v", resp6.StatusCode, err)
		}

		searchReq := httptest.NewRequest(http.MethodPost, "/appointments/search/clinics", strings.NewReader(url.Values{"searchTerm": {""}}.Encode()))
		searchReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp7, err := app.Test(searchReq)
		if err != nil || resp7.StatusCode != fiber.StatusOK {
			t.Fatalf("search clinics expected 200, got status=%d err=%v", resp7.StatusCode, err)
		}

		resp8, err := app.Test(httptest.NewRequest(http.MethodPost, "/appointments/"+apnt.ID+"/delete", nil))
		if err != nil || resp8.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("delete expected 303, got status=%d err=%v", resp8.StatusCode, err)
		}
	})

	t.Run("patient token routes", func(t *testing.T) {
		tokenApnt := models.Appointment{
			UserID:    user.ID,
			ClinicID:  clinic.ID,
			PatientID: patient.ID,
			BookedAt:  time.Now().Add(5 * time.Hour),
			Token:     "tok_patient_routes",
		}
		if err := svc.CreateAppointment(t.Context(), &tokenApnt); err != nil {
			t.Fatalf("seed token appointment: %v", err)
		}

		app := fiber.New()
		app.Get("/appointments/:id/patient/confirm/:token", svc.HandlePatientConfirm)
		app.Get("/appointments/:id/patient/cancel/:token", svc.HandlePatientCancelPage)
		app.Post("/appointments/:id/patient/cancel/:token", svc.HandlePatientCancel)
		app.Get("/appointments/:id/patient/changedate/:token", svc.HandlePatientChangeDate)

		resp1, err := app.Test(httptest.NewRequest(http.MethodGet, "/appointments/"+tokenApnt.ID+"/patient/confirm/"+tokenApnt.Token, nil))
		if err != nil || resp1.StatusCode != fiber.StatusOK {
			t.Fatalf("patient confirm expected 200, got status=%d err=%v", resp1.StatusCode, err)
		}

		resp2, err := app.Test(httptest.NewRequest(http.MethodGet, "/appointments/"+tokenApnt.ID+"/patient/cancel/"+tokenApnt.Token, nil))
		if err != nil || resp2.StatusCode != fiber.StatusOK {
			t.Fatalf("patient cancel page expected 200, got status=%d err=%v", resp2.StatusCode, err)
		}

		resp3, err := app.Test(httptest.NewRequest(http.MethodPost, "/appointments/"+tokenApnt.ID+"/patient/cancel/"+tokenApnt.Token, nil))
		if err != nil || resp3.StatusCode != fiber.StatusOK {
			t.Fatalf("patient cancel expected 200, got status=%d err=%v", resp3.StatusCode, err)
		}

		resp4, err := app.Test(httptest.NewRequest(http.MethodGet, "/appointments/"+tokenApnt.ID+"/patient/changedate/"+tokenApnt.Token, nil))
		if err != nil || resp4.StatusCode != fiber.StatusOK {
			t.Fatalf("patient change date expected 200, got status=%d err=%v", resp4.StatusCode, err)
		}
	})
}

func TestAppointmentHandlerGuardBranches(t *testing.T) {
	svc, user, _, _ := newAppointmentServiceForTests(t)
	app := fiber.New()
	app.Get("/appointments/start", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleStartPage(c)
	})
	app.Get("/appointments/new/pricefrg/:id", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandlePriceFrg(c)
	})
	app.Post("/appointments", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleCreate(c)
	})
	app.Post("/appointments/:id/complete", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleComplete(c)
	})
	app.Post("/appointments/:id/patch", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleUpdate(c)
	})
	app.Post("/appointments/:id/cancel", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleCancel(c)
	})
	app.Post("/appointments/:id/delete", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleDelete(c)
	})
	app.Post("/appointments/search/clinics", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleSearchClinics(c)
	})

	resp1, err := app.Test(httptest.NewRequest(http.MethodGet, "/appointments/new/pricefrg/", nil))
	if err != nil || resp1.StatusCode != fiber.StatusNotFound {
		t.Fatalf("price fragment missing id expected 404, got status=%d err=%v", resp1.StatusCode, err)
	}

	respStart, err := app.Test(httptest.NewRequest(http.MethodGet, "/appointments/start", nil))
	if err != nil || respStart.StatusCode != fiber.StatusSeeOther {
		t.Fatalf("start without id expected 303 redirect, got status=%d err=%v", respStart.StatusCode, err)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/appointments", strings.NewReader("{"))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("HX-Request", "true")
	respCreate, err := app.Test(createReq)
	if err != nil || respCreate.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("create invalid body expected 400 for htmx, got status=%d err=%v", respCreate.StatusCode, err)
	}

	resp2, err := app.Test(httptest.NewRequest(http.MethodPost, "/appointments/%20/complete", nil))
	if err != nil || resp2.StatusCode != fiber.StatusSeeOther {
		t.Fatalf("complete blank id expected 303 redirect, got status=%d err=%v", resp2.StatusCode, err)
	}

	resp3, err := app.Test(httptest.NewRequest(http.MethodPost, "/appointments/%20/patch", nil))
	if err != nil || resp3.StatusCode != fiber.StatusSeeOther {
		t.Fatalf("update blank id expected 303 redirect, got status=%d err=%v", resp3.StatusCode, err)
	}

	resp4, err := app.Test(httptest.NewRequest(http.MethodPost, "/appointments/%20/cancel", nil))
	if err != nil || resp4.StatusCode != fiber.StatusSeeOther {
		t.Fatalf("cancel blank id expected 303 redirect, got status=%d err=%v", resp4.StatusCode, err)
	}

	resp5, err := app.Test(httptest.NewRequest(http.MethodPost, "/appointments/%20/delete", nil))
	if err != nil || resp5.StatusCode != fiber.StatusSeeOther {
		t.Fatalf("delete blank id expected 303 redirect, got status=%d err=%v", resp5.StatusCode, err)
	}

	searchReq := httptest.NewRequest(http.MethodPost, "/appointments/search/clinics", strings.NewReader(url.Values{"searchTerm": {"alpha"}}.Encode()))
	searchReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp6, err := app.Test(searchReq)
	if err != nil || resp6.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("search clinics with FTS expected 500, got status=%d err=%v", resp6.StatusCode, err)
	}
}

func TestHandleStartPagePatientMissingBranch(t *testing.T) {
	svc, user, clinic, _ := newAppointmentServiceForTests(t)
	apnt := models.Appointment{
		UserID:    user.ID,
		ClinicID:  clinic.ID,
		PatientID: "missing-patient-id",
		BookedAt:  time.Now().Add(time.Hour),
		Token:     "tok_missing_patient",
	}
	if err := svc.CreateAppointment(t.Context(), &apnt); err != nil {
		t.Fatalf("create appointment: %v", err)
	}

	app := fiber.New()
	app.Get("/appointments/:id/start", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleStartPage(c)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/appointments/"+apnt.ID+"/start", nil))
	if err != nil || resp.StatusCode != fiber.StatusSeeOther {
		t.Fatalf("start with missing patient expected 303 redirect, got status=%d err=%v", resp.StatusCode, err)
	}
}
