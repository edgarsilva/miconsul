package patient

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
)

func TestPatientHandlerGuardBranches(t *testing.T) {
	svc, user := newPatientServiceForTests(t)

	app := fiber.New()
	app.Get("/patients/:id", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandlePatientFormPage(c)
	})
	app.Post("/patients", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleCreatePatient(c)
	})
	app.Post("/patients/:id/patch", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleUpdatePatient(c)
	})
	app.Post("/patients/:id/removepic", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleRemovePic(c)
	})
	app.Post("/patients/:id/delete", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleDeletePatient(c)
	})
	app.Get("/patients/search", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandlePatientsIndexSearch(c)
	})
	app.Post("/patients/search", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandlePatientSearch(c)
	})
	app.Get("/patients/:id/profilepic/:filename", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandlePatientProfilePicImgSrc(c)
	})

	resp1, err := app.Test(httptest.NewRequest(http.MethodGet, "/patients/%20", nil))
	if err != nil || resp1.StatusCode != fiber.StatusSeeOther {
		t.Fatalf("patient form blank id expected 303, got status=%d err=%v", resp1.StatusCode, err)
	}

	badJSON := httptest.NewRequest(http.MethodPost, "/patients", strings.NewReader("{"))
	badJSON.Header.Set("Content-Type", "application/json")
	resp2, err := app.Test(badJSON)
	if err != nil || resp2.StatusCode != fiber.StatusSeeOther {
		t.Fatalf("create invalid body expected 303 redirect, got status=%d err=%v", resp2.StatusCode, err)
	}

	resp3, err := app.Test(httptest.NewRequest(http.MethodPost, "/patients//patch", nil))
	if err != nil || resp3.StatusCode != fiber.StatusNotFound {
		t.Fatalf("patch missing route id expected 404, got status=%d err=%v", resp3.StatusCode, err)
	}

	resp4, err := app.Test(httptest.NewRequest(http.MethodPost, "/patients//removepic", nil))
	if err != nil || resp4.StatusCode != fiber.StatusNotFound {
		t.Fatalf("remove pic missing route id expected 404, got status=%d err=%v", resp4.StatusCode, err)
	}

	resp5, err := app.Test(httptest.NewRequest(http.MethodPost, "/patients//delete", nil))
	if err != nil || resp5.StatusCode != fiber.StatusNotFound {
		t.Fatalf("delete missing route id expected 404, got status=%d err=%v", resp5.StatusCode, err)
	}

	resp6, err := app.Test(httptest.NewRequest(http.MethodGet, "/patients/search?searchTerm=ab", nil))
	if err != nil || resp6.StatusCode != fiber.StatusSeeOther {
		t.Fatalf("index search short term expected 303, got status=%d err=%v", resp6.StatusCode, err)
	}

	shortForm := url.Values{"searchTerm": {"ab"}}
	req7 := httptest.NewRequest(http.MethodPost, "/patients/search", strings.NewReader(shortForm.Encode()))
	req7.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp7, err := app.Test(req7)
	if err != nil || resp7.StatusCode != fiber.StatusSeeOther {
		t.Fatalf("patient search short term expected 303, got status=%d err=%v", resp7.StatusCode, err)
	}

	resp8, err := app.Test(httptest.NewRequest(http.MethodGet, "/patients//profilepic/", nil))
	if err != nil || resp8.StatusCode != fiber.StatusNotFound {
		t.Fatalf("profile pic missing params expected 404, got status=%d err=%v", resp8.StatusCode, err)
	}
}

func TestHandlePatientSearchKeepsSelection(t *testing.T) {
	svc, user := newPatientServiceForTests(t)

	patients := make([]models.Patient, QUERY_LIMIT+1)
	for i := range patients {
		patients[i] = models.Patient{UserID: user.ID, Name: fmt.Sprintf("Patient %d", i), Age: 30, Phone: "555", Email: "p@example.com"}
		if err := svc.DB.GormDB().Create(&patients[i]).Error; err != nil {
			t.Fatalf("create patient: %v", err)
		}
	}
	oldest, newest := patients[0], patients[len(patients)-1]

	app := fiber.New()
	app.Post("/patients/search", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandlePatientSearch(c)
	})

	tests := []struct {
		name        string
		selected    string
		wantChecked string
	}{
		{"selected patient in results stays checked", patients[5].UID, patients[5].UID},
		{"selected patient past the limit is listed and checked", oldest.UID, oldest.UID},
		{"no selection checks the first patient", "", newest.UID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{"searchTerm": {""}}
			if tt.selected != "" {
				form.Set("patientId", tt.selected)
			}
			req := httptest.NewRequest(http.MethodPost, "/patients/search", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp, err := app.Test(req)
			if err != nil || resp.StatusCode != fiber.StatusOK {
				t.Fatalf("search patients expected 200, got status=%d err=%v", resp.StatusCode, err)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}

			inputs := regexp.MustCompile(`<input[^>]*name="patientId"[^>]*>`).FindAllString(string(body), -1)
			checked := []string{}
			for _, input := range inputs {
				if strings.Contains(input, " checked") {
					checked = append(checked, regexp.MustCompile(`value="([^"]*)"`).FindStringSubmatch(input)[1])
				}
			}
			if len(checked) != 1 || checked[0] != tt.wantChecked {
				t.Errorf("checked patients = %v, want [%s]", checked, tt.wantChecked)
			}
		})
	}
}
