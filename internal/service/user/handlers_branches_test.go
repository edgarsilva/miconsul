package user

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"miconsul/internal/model"

	"github.com/gofiber/fiber/v3"
)

func TestUserHandlerBranches(t *testing.T) {
	svc, currentUser := newUserServiceForTests(t)

	app := fiber.New()
	app.Get("/admin/users/:id", func(c fiber.Ctx) error {
		return svc.HandleEditPage(c)
	})
	app.Post("/profile", func(c fiber.Ctx) error {
		c.Locals("current_user", currentUser)
		return svc.HandleUpdateProfile(c)
	})
	app.Get("/api/users", func(c fiber.Ctx) error {
		return svc.HandleAPIUsers(c)
	})
	app.Post("/api/users/make/:n", func(c fiber.Ctx) error {
		return svc.HandleAPIMakeUsers(c)
	})

	resp1, err := app.Test(httptest.NewRequest(http.MethodGet, "/admin/users/%20", nil))
	if err != nil || resp1.StatusCode != fiber.StatusSeeOther {
		t.Fatalf("edit page blank id expected 303, got status=%d err=%v", resp1.StatusCode, err)
	}

	badJSON := httptest.NewRequest(http.MethodPost, "/profile", strings.NewReader("{"))
	badJSON.Header.Set("Content-Type", "application/json")
	badJSON.Header.Set("HX-Request", "true")
	resp2, err := app.Test(badJSON)
	if err != nil || resp2.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("profile invalid body expected 400, got status=%d err=%v", resp2.StatusCode, err)
	}
	if got := resp2.Header.Get("HX-Location"); got == "" {
		t.Fatalf("expected HX-Location header for htmx profile error")
	}

	resp3, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/users", nil))
	if err != nil || resp3.StatusCode != fiber.StatusOK {
		t.Fatalf("api users expected 200, got status=%d err=%v", resp3.StatusCode, err)
	}

	resp4, err := app.Test(httptest.NewRequest(http.MethodPost, "/api/users/make/1", nil))
	if err != nil || resp4.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("api make users expected 422 on db constraints, got status=%d err=%v", resp4.StatusCode, err)
	}
	body, _ := io.ReadAll(resp4.Body)
	if !strings.Contains(string(body), "unprocessable entity") {
		t.Fatalf("expected unprocessable entity payload, got %q", string(body))
	}
}

func TestHandleEditPageSuccess(t *testing.T) {
	svc, currentUser := newUserServiceForTests(t)

	admin := model.User{Email: "admin@example.com", Password: "hash", Role: model.UserRoleAdmin}
	if err := svc.CreateUsersInBatches(t.Context(), []model.User{admin}, 1); err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	app := fiber.New()
	app.Get("/admin/users/:id", func(c fiber.Ctx) error {
		return svc.HandleEditPage(c)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/admin/users/"+currentUser.ID, nil))
	if err != nil || resp.StatusCode != fiber.StatusOK {
		t.Fatalf("edit page existing user expected 200, got status=%d err=%v", resp.StatusCode, err)
	}
}
