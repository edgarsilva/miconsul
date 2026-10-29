package patient

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

	resp8, err := app.Test(httptest.NewRequest(http.MethodGet, "/patients//profilepic/", nil))
	if err != nil || resp8.StatusCode != fiber.StatusNotFound {
		t.Fatalf("profile pic missing params expected 404, got status=%d err=%v", resp8.StatusCode, err)
	}
}
