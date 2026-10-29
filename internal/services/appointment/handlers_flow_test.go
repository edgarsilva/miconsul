package appointment

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
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
		app.Get("/appointments/search", func(c fiber.Ctx) error {
			c.Locals("current_user", user)
			return svc.HandleIndexSearch(c)
		})
		app.Get("/appointments/:id", func(c fiber.Ctx) error {
			c.Locals("current_user", user)
			return svc.HandleShowPage(c)
		})

		resp1, err := app.Test(httptest.NewRequest(http.MethodGet, "/appointments", nil))
		if err != nil || resp1.StatusCode != fiber.StatusOK {
			t.Fatalf("index page expected 200, got status=%d err=%v", resp1.StatusCode, err)
		}

		resp2, err := app.Test(httptest.NewRequest(http.MethodGet, "/appointments/"+apnt.UID, nil))
		if err != nil || resp2.StatusCode != fiber.StatusOK {
			t.Fatalf("show page expected 200, got status=%d err=%v", resp2.StatusCode, err)
		}

		resp3, err := app.Test(httptest.NewRequest(http.MethodGet, "/appointments/search?searchTerm=", nil))
		if err != nil || resp3.StatusCode != fiber.StatusOK {
			t.Fatalf("search page expected 200, got status=%d err=%v", resp3.StatusCode, err)
		}
	})

	t.Run("start page and create/update/delete endpoints", func(t *testing.T) {
		app := fiber.New()
		app.Get("/appointments/:id/open", func(c fiber.Ctx) error {
			c.Locals("current_user", user)
			return svc.HandleOpenPage(c)
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

		resp1, err := app.Test(httptest.NewRequest(http.MethodGet, "/appointments/"+apnt.UID+"/open", nil))
		if err != nil || resp1.StatusCode != fiber.StatusOK {
			t.Fatalf("start page expected 200, got status=%d err=%v", resp1.StatusCode, err)
		}

		form := url.Values{
			"bookedAt":  {time.Now().Format("2006-01-02T15:04")},
			"price":     {"100.0"},
			"clinicId":  {clinic.UID},
			"patientId": {patient.UID},
			"duration":  {"30"},
		}
		req2 := httptest.NewRequest(http.MethodPost, "/appointments", strings.NewReader(form.Encode()))
		req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp2, err := app.Test(req2)
		if err != nil || resp2.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("create expected 303, got status=%d err=%v", resp2.StatusCode, err)
		}

		var created models.Appointment
		if err := svc.DB.WithContext(t.Context()).Where("user_id = ? AND uid <> ?", user.ID, apnt.UID).Order("id desc").Take(&created).Error; err != nil {
			t.Fatalf("load created appointment: %v", err)
		}
		// The UID's first 16 xid chars encode time, machine and pid.
		uidBody := strings.TrimPrefix(created.UID, "apnt")
		if created.Token == "" || strings.Contains(created.Token, uidBody[:16]) {
			t.Fatalf("token must not be derived from the UID: uid=%q token=%q", created.UID, created.Token)
		}

		updForm := url.Values{
			"bookedAt":  {time.Now().Add(2 * time.Hour).Format("2006-01-02T15:04")},
			"price":     {"120.0"},
			"clinicId":  {clinic.UID},
			"patientId": {patient.UID},
			"duration":  {"45"},
		}
		req3 := httptest.NewRequest(http.MethodPost, "/appointments/"+apnt.UID+"/patch", strings.NewReader(updForm.Encode()))
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
		req4 := httptest.NewRequest(http.MethodPost, "/appointments/"+apnt.UID+"/complete", strings.NewReader(completeForm.Encode()))
		req4.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp4, err := app.Test(req4)
		if err != nil || resp4.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("complete expected 303, got status=%d err=%v", resp4.StatusCode, err)
		}

		resp5, err := app.Test(httptest.NewRequest(http.MethodPost, "/appointments/"+apnt.UID+"/cancel", nil))
		if err != nil || resp5.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("cancel expected 303, got status=%d err=%v", resp5.StatusCode, err)
		}

		resp6, err := app.Test(httptest.NewRequest(http.MethodGet, "/appointments/new/pricefrg/"+clinic.UID, nil))
		if err != nil || resp6.StatusCode != fiber.StatusOK {
			t.Fatalf("price fragment expected 200, got status=%d err=%v", resp6.StatusCode, err)
		}

		searchReq := httptest.NewRequest(http.MethodPost, "/appointments/search/clinics", strings.NewReader(url.Values{"searchTerm": {""}}.Encode()))
		searchReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp7, err := app.Test(searchReq)
		if err != nil || resp7.StatusCode != fiber.StatusOK {
			t.Fatalf("search clinics expected 200, got status=%d err=%v", resp7.StatusCode, err)
		}

		resp8, err := app.Test(httptest.NewRequest(http.MethodPost, "/appointments/"+apnt.UID+"/delete", nil))
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
		app.Get("/p/a/:token/confirm", svc.HandlePatientConfirmPage)
		app.Post("/p/a/:token/confirm", svc.HandlePatientConfirm)
		app.Get("/p/a/:token/cancel", svc.HandlePatientCancelPage)
		app.Post("/p/a/:token/cancel", svc.HandlePatientCancel)
		app.Post("/p/a/:token/changedate", svc.HandlePatientChangeDate)

		confirmPath := tokenApnt.ConfirmPath()
		resp0, err := app.Test(httptest.NewRequest(http.MethodGet, confirmPath, nil))
		if err != nil || resp0.StatusCode != fiber.StatusOK {
			t.Fatalf("patient confirm page expected 200, got status=%d err=%v", resp0.StatusCode, err)
		}
		afterPage, err := svc.TakeAppointmentByToken(t.Context(), tokenApnt.Token)
		if err != nil {
			t.Fatalf("load appointment after confirm page: %v", err)
		}
		if !afterPage.ConfirmedAt.IsZero() {
			t.Fatalf("GET confirm page must not confirm the appointment")
		}

		resp1, err := app.Test(httptest.NewRequest(http.MethodPost, confirmPath, nil))
		if err != nil || resp1.StatusCode != fiber.StatusOK {
			t.Fatalf("patient confirm expected 200, got status=%d err=%v", resp1.StatusCode, err)
		}
		afterPost, err := svc.TakeAppointmentByToken(t.Context(), tokenApnt.Token)
		if err != nil {
			t.Fatalf("load appointment after confirm: %v", err)
		}
		if afterPost.ConfirmedAt.IsZero() {
			t.Fatalf("POST confirm must confirm the appointment")
		}

		resp2, err := app.Test(httptest.NewRequest(http.MethodGet, tokenApnt.CancelPath(), nil))
		if err != nil || resp2.StatusCode != fiber.StatusOK {
			t.Fatalf("patient cancel page expected 200, got status=%d err=%v", resp2.StatusCode, err)
		}

		resp3, err := app.Test(httptest.NewRequest(http.MethodPost, tokenApnt.CancelPath(), nil))
		if err != nil || resp3.StatusCode != fiber.StatusOK {
			t.Fatalf("patient cancel expected 200, got status=%d err=%v", resp3.StatusCode, err)
		}

		resp4, err := app.Test(httptest.NewRequest(http.MethodPost, "/p/a/"+tokenApnt.Token+"/changedate", nil))
		if err != nil || resp4.StatusCode != fiber.StatusConflict {
			t.Fatalf("patient change date after cancel expected 409, got status=%d err=%v", resp4.StatusCode, err)
		}
	})
}

func TestAppointmentHandlerGuardBranches(t *testing.T) {
	svc, user, _, _ := newAppointmentServiceForTests(t)
	app := fiber.New()
	app.Get("/appointments/open", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleOpenPage(c)
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

	respStart, err := app.Test(httptest.NewRequest(http.MethodGet, "/appointments/open", nil))
	if err != nil || respStart.StatusCode != fiber.StatusSeeOther {
		t.Fatalf("start without id expected 303 redirect, got status=%d err=%v", respStart.StatusCode, err)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/appointments", strings.NewReader("{"))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("HX-Request", "true")
	respCreate, err := app.Test(createReq)
	if err != nil || respCreate.StatusCode != fiber.StatusNoContent {
		t.Fatalf("create invalid body expected 204 for htmx redirect, got status=%d err=%v", respCreate.StatusCode, err)
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
		PatientID: 999999,
		BookedAt:  time.Now().Add(time.Hour),
		Token:     "tok_missing_patient",
	}
	if err := svc.CreateAppointment(t.Context(), &apnt); err != nil {
		t.Fatalf("create appointment: %v", err)
	}

	app := fiber.New()
	app.Get("/appointments/:id/open", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleOpenPage(c)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/appointments/"+apnt.UID+"/open", nil))
	if err != nil || resp.StatusCode != fiber.StatusSeeOther {
		t.Fatalf("start with missing patient expected 303 redirect, got status=%d err=%v", resp.StatusCode, err)
	}
}

func TestHandleSearchClinicsKeepsSelection(t *testing.T) {
	svc, user, _, _ := newAppointmentServiceForTests(t)

	extra := make([]models.Clinic, 11)
	for i := range extra {
		extra[i] = models.Clinic{UserID: user.ID, Name: fmt.Sprintf("Clinic %d", i), Email: "c@example.com", Phone: "123"}
		if err := svc.DB.GormDB().Create(&extra[i]).Error; err != nil {
			t.Fatalf("create clinic: %v", err)
		}
	}
	oldest := extra[0]

	app := fiber.New()
	app.Post("/appointments/search/clinics", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleSearchClinics(c)
	})

	tests := []struct {
		name        string
		selected    string
		wantChecked string
		wantPrice   bool
	}{
		{"selected clinic in results stays checked", extra[2].UID, extra[2].UID, false},
		{"selected clinic past the limit is listed and checked", oldest.UID, oldest.UID, false},
		{"no selection checks the first clinic and sets its price", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{"searchTerm": {""}}
			if tt.selected != "" {
				form.Set("clinicId", tt.selected)
			}
			req := httptest.NewRequest(http.MethodPost, "/appointments/search/clinics", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp, err := app.Test(req)
			if err != nil || resp.StatusCode != fiber.StatusOK {
				t.Fatalf("search clinics expected 200, got status=%d err=%v", resp.StatusCode, err)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}

			checked := checkedRadios(string(body), "clinicId")
			if len(checked) != 1 {
				t.Fatalf("checked clinics = %v, want exactly one", checked)
			}
			if tt.wantChecked != "" && checked[0] != tt.wantChecked {
				t.Errorf("checked clinic = %q, want %q", checked[0], tt.wantChecked)
			}
			if got := strings.Contains(string(body), `id="appt_price"`); got != tt.wantPrice {
				t.Errorf("price swapped = %v, want %v", got, tt.wantPrice)
			}
		})
	}
}

// checkedRadios returns the values of the checked radio inputs named name.
func checkedRadios(html, name string) []string {
	inputs := regexp.MustCompile(`<input[^>]*name="`+name+`"[^>]*>`).FindAllString(html, -1)
	value := regexp.MustCompile(`value="([^"]*)"`)

	checked := []string{}
	for _, input := range inputs {
		if strings.Contains(input, " checked") {
			checked = append(checked, value.FindStringSubmatch(input)[1])
		}
	}

	return checked
}

func TestHandleSearchPatientsKeepsSelection(t *testing.T) {
	svc, user, _, _ := newAppointmentServiceForTests(t)

	extra := make([]models.Patient, 11)
	for i := range extra {
		extra[i] = models.Patient{UserID: user.ID, Name: fmt.Sprintf("Patient %d", i), Age: 30, Phone: "555", Email: "p@example.com"}
		if err := svc.DB.GormDB().Create(&extra[i]).Error; err != nil {
			t.Fatalf("create patient: %v", err)
		}
	}
	oldest, newest := extra[0], extra[len(extra)-1]

	app := fiber.New()
	app.Post("/appointments/search/patients", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleSearchPatients(c)
	})

	tests := []struct {
		name        string
		selected    string
		wantChecked string
	}{
		{"selected patient in results stays checked", extra[5].UID, extra[5].UID},
		{"selected patient past the limit is listed and checked", oldest.UID, oldest.UID},
		{"no selection checks the first patient", "", newest.UID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{"searchTerm": {""}}
			if tt.selected != "" {
				form.Set("patientId", tt.selected)
			}
			req := httptest.NewRequest(http.MethodPost, "/appointments/search/patients", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp, err := app.Test(req)
			if err != nil || resp.StatusCode != fiber.StatusOK {
				t.Fatalf("search patients expected 200, got status=%d err=%v", resp.StatusCode, err)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}

			checked := checkedRadios(string(body), "patientId")
			if len(checked) != 1 || checked[0] != tt.wantChecked {
				t.Errorf("checked patients = %v, want [%s]", checked, tt.wantChecked)
			}
		})
	}
}

func TestPickerSearchToasts(t *testing.T) {
	svc, user, _, _ := newAppointmentServiceForTests(t)

	app := fiber.New()
	app.Post("/appointments/search/clinics", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleSearchClinics(c)
	})
	app.Post("/appointments/search/patients", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleSearchPatients(c)
	})

	tests := []struct {
		name       string
		path       string
		radio      string
		term       string
		wantStatus int
		wantToast  string
	}{
		{"clinic short term", "/appointments/search/clinics", "clinicId", "ab", fiber.StatusOK, "at least 3 characters"},
		{"patient short term", "/appointments/search/patients", "patientId", "ab", fiber.StatusOK, "at least 3 characters"},
		// The test DB has no FTS table, so any real search term fails.
		{"clinic search error", "/appointments/search/clinics", "clinicId", "alpha", fiber.StatusInternalServerError, "couldn&#39;t be completed"},
		{"patient search error", "/appointments/search/patients", "patientId", "alpha", fiber.StatusInternalServerError, "couldn&#39;t be completed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(url.Values{"searchTerm": {tt.term}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			resp, err := app.Test(req)
			if err != nil || resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got status=%d err=%v", tt.wantStatus, resp.StatusCode, err)
			}
			if got := resp.Header.Get("HX-Redirect"); got != "" {
				t.Errorf("search redirected to %q", got)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if !strings.Contains(string(body), `<hx-partial hx-target="#toast_region"`) || !strings.Contains(string(body), tt.wantToast) {
				t.Errorf("body = %s, want only a toast partial with %q", body, tt.wantToast)
			}
			if strings.Contains(string(body), `name="`+tt.radio+`"`) {
				t.Errorf("search replaced the %s list", tt.radio)
			}
		})
	}
}
