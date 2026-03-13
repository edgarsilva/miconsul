package clinic

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"miconsul/internal/model"

	"github.com/gofiber/fiber/v3"
)

func TestClinicHandlerBranches(t *testing.T) {
	svc, user := newClinicServiceForTests(t)

	app := fiber.New()
	app.Get("/clinics/search", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleClinicsIndexSearch(c)
	})
	app.Get("/clinics/:id", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleClinicsShowPage(c)
	})
	app.Post("/clinics", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleClinicsCreate(c)
	})
	app.Post("/clinics/:id/patch", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleClinicsUpdate(c)
	})
	app.Post("/clinics/:id/delete", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleClinicsDelete(c)
	})

	resp1, err := app.Test(httptest.NewRequest(http.MethodGet, "/clinics/%20", nil))
	if err != nil || resp1.StatusCode != fiber.StatusNotFound {
		t.Fatalf("show with missing clinic expected 404, got status=%d err=%v", resp1.StatusCode, err)
	}

	resp2, err := app.Test(httptest.NewRequest(http.MethodGet, "/clinics/search?searchTerm=ab", nil))
	if err != nil || resp2.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("search short term expected 400, got status=%d err=%v", resp2.StatusCode, err)
	}

	badJSON := httptest.NewRequest(http.MethodPost, "/clinics", strings.NewReader("{"))
	badJSON.Header.Set("Content-Type", "application/json")
	badJSON.Header.Set("HX-Request", "true")
	resp3, err := app.Test(badJSON)
	if err != nil || resp3.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("create invalid body expected 400 for htmx, got status=%d err=%v", resp3.StatusCode, err)
	}

	resp4, err := app.Test(httptest.NewRequest(http.MethodPost, "/clinics/%20/patch", nil))
	if err != nil || resp4.StatusCode != fiber.StatusNotFound {
		t.Fatalf("update missing clinic expected 404, got status=%d err=%v", resp4.StatusCode, err)
	}

	resp5, err := app.Test(httptest.NewRequest(http.MethodPost, "/clinics/%20/delete", nil))
	if err != nil || resp5.StatusCode != fiber.StatusNotFound {
		t.Fatalf("delete missing clinic expected 404, got status=%d err=%v", resp5.StatusCode, err)
	}
}

func TestClinicHandlersHappyPaths(t *testing.T) {
	svc, user := newClinicServiceForTests(t)

	clinic := model.Clinic{UserID: user.ID, Name: "Alpha", Email: "alpha@example.com", Phone: "123"}
	if err := svc.CreateClinic(t.Context(), &clinic); err != nil {
		t.Fatalf("seed clinic: %v", err)
	}

	app := fiber.New()
	app.Get("/clinics/:id", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleClinicsShowPage(c)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/clinics/"+clinic.ID, nil))
	if err != nil || resp.StatusCode != fiber.StatusOK {
		t.Fatalf("show existing clinic expected 200, got status=%d err=%v", resp.StatusCode, err)
	}
}
