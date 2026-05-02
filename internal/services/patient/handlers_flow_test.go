package patient

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func TestPatientHandlersFlows(t *testing.T) {
	svc, user := newPatientServiceForTests(t)

	seed := models.Patient{UserID: user.ID, Name: "Alpha", Age: 30, Phone: "111", Email: "alpha@example.com"}
	if err := svc.CreatePatient(t.Context(), &seed); err != nil {
		t.Fatalf("seed patient: %v", err)
	}

	app := fiber.New()
	app.Get("/patients/search", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandlePatientsIndexSearch(c)
	})
	app.Get("/patients", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleIndexPage(c)
	})
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
	app.Post("/patients/search", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandlePatientSearch(c)
	})

	resp1, err := app.Test(httptest.NewRequest(http.MethodGet, "/patients", nil))
	if err != nil || resp1.StatusCode != fiber.StatusOK {
		t.Fatalf("index page expected 200, got status=%d err=%v", resp1.StatusCode, err)
	}

	resp2, err := app.Test(httptest.NewRequest(http.MethodGet, "/patients/"+seed.UID, nil))
	if err != nil || resp2.StatusCode != fiber.StatusOK {
		t.Fatalf("patient form page expected 200, got status=%d err=%v", resp2.StatusCode, err)
	}

	createForm := url.Values{
		"name":  {"Created Patient"},
		"email": {"created@example.com"},
		"phone": {"999"},
		"age":   {"22"},
	}
	req3 := httptest.NewRequest(http.MethodPost, "/patients", strings.NewReader(createForm.Encode()))
	req3.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp3, err := app.Test(req3)
	if err != nil || resp3.StatusCode != fiber.StatusSeeOther {
		t.Fatalf("create patient expected 303, got status=%d err=%v", resp3.StatusCode, err)
	}

	updateForm := url.Values{
		"name":  {"Alpha Updated"},
		"email": {"updated@example.com"},
		"phone": {"444"},
		"age":   {"31"},
	}
	req4 := httptest.NewRequest(http.MethodPost, "/patients/"+seed.UID+"/patch", strings.NewReader(updateForm.Encode()))
	req4.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req4.Header.Set("HX-Request", "true")
	resp4, err := app.Test(req4)
	if err != nil || resp4.StatusCode != fiber.StatusOK {
		t.Fatalf("update patient expected 200 for htmx, got status=%d err=%v", resp4.StatusCode, err)
	}

	req5 := httptest.NewRequest(http.MethodPost, "/patients/"+seed.UID+"/removepic", nil)
	req5.Header.Set("HX-Request", "true")
	resp5, err := app.Test(req5)
	if err != nil || resp5.StatusCode != fiber.StatusOK {
		t.Fatalf("remove pic expected 200 for htmx, got status=%d err=%v", resp5.StatusCode, err)
	}

	req6 := httptest.NewRequest(http.MethodPost, "/patients/"+seed.UID+"/delete", nil)
	req6.Header.Set("HX-Request", "true")
	resp6, err := app.Test(req6)
	if err != nil || resp6.StatusCode != fiber.StatusOK {
		t.Fatalf("delete patient expected 200 for htmx, got status=%d err=%v", resp6.StatusCode, err)
	}

	postSearch := httptest.NewRequest(http.MethodPost, "/patients/search", strings.NewReader(url.Values{"searchTerm": {""}}.Encode()))
	postSearch.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp7, err := app.Test(postSearch)
	if err != nil || resp7.StatusCode != fiber.StatusOK {
		t.Fatalf("patient search expected 200, got status=%d err=%v", resp7.StatusCode, err)
	}

	resp8, err := app.Test(httptest.NewRequest(http.MethodGet, "/patients/search?searchTerm=", nil))
	if err != nil || resp8.StatusCode != fiber.StatusOK {
		t.Fatalf("index search expected 200, got status=%d err=%v", resp8.StatusCode, err)
	}
}

func TestPatientProfilePicImageSourceHandler(t *testing.T) {
	svc, user := newPatientServiceForTests(t)

	assetsDir := t.TempDir()
	svc.Env.AssetsDir = assetsDir

	patient := models.Patient{UserID: user.ID, Name: "Pic User", Age: 28, Phone: "123", Email: "pic@example.com"}
	if err := svc.CreatePatient(t.Context(), &patient); err != nil {
		t.Fatalf("create patient: %v", err)
	}

	filename := patient.UID + "_ppic_avatar.png"
	storagePath, err := ProfilePicPath(filename, assetsDir)
	if err != nil {
		t.Fatalf("profile pic path: %v", err)
	}
	if err := os.WriteFile(storagePath, []byte("fake-image"), 0o644); err != nil {
		t.Fatalf("write profile pic file: %v", err)
	}

	if err := svc.DB.WithContext(t.Context()).Model(&models.Patient{}).Where("id = ?", patient.ID).Update("profile_pic", "/patients/"+patient.UID+"/profilepic/"+filename).Error; err != nil {
		t.Fatalf("attach patient profile pic: %v", err)
	}

	app := fiber.New()
	app.Get("/patients/:id/profilepic/:filename", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandlePatientProfilePicImgSrc(c)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/patients/"+patient.UID+"/profilepic/"+filename, nil))
	if err != nil || resp.StatusCode != fiber.StatusOK {
		t.Fatalf("profile pic src expected 200, got status=%d err=%v", resp.StatusCode, err)
	}

	var count int64
	if err := svc.DB.WithContext(t.Context()).Model(&models.Patient{}).Where("id = ?", patient.ID).Count(&count).Error; err != nil {
		t.Fatalf("count patient rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected patient row to remain present, got %d", count)
	}

	if _, err := os.Stat(filepath.Dir(storagePath)); err != nil {
		t.Fatalf("expected profile pic directory to exist: %v", err)
	}
}

func TestDeletePatientMissingIDGuardRoute(t *testing.T) {
	svc, user := newPatientServiceForTests(t)

	app := fiber.New()
	app.Post("/patients/delete", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleDeletePatient(c)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodPost, "/patients/delete", nil))
	if err != nil || resp.StatusCode != fiber.StatusSeeOther {
		t.Fatalf("missing id delete guard expected 303 redirect, got status=%d err=%v", resp.StatusCode, err)
	}
}

func TestTakePatientByIDNotFound(t *testing.T) {
	svc, user := newPatientServiceForTests(t)

	_, err := svc.TakePatientByID(t.Context(), user.ID, "missing")
	if err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected gorm not found for missing patient, got %v", err)
	}
}
