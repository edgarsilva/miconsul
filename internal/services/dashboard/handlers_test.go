package dashboard

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/models"
	"miconsul/internal/server"

	"github.com/gofiber/fiber/v3"
)

func TestHandleHomePage(t *testing.T) {
	t.Run("renders public landing for anonymous users", func(t *testing.T) {
		svc, err := NewService(nil)
		if err == nil {
			t.Fatalf("expected error creating dashboard service with nil server")
		}

		svc, err = NewService(newDashboardServerForTests())
		if err != nil {
			t.Fatalf("new dashboard service: %v", err)
		}

		app := fiber.New()
		app.Get("/", svc.HandleHomePage)

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("redirects logged in users to dashboard", func(t *testing.T) {
		svc, err := NewService(newDashboardServerForTests())
		if err != nil {
			t.Fatalf("new dashboard service: %v", err)
		}

		app := fiber.New()
		app.Get("/", func(c fiber.Ctx) error {
			c.Locals("current_user", models.User{UID: "user_1"})
			return svc.HandleHomePage(c)
		})

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); got != "/dashboard?timeframe=day" {
			t.Fatalf("expected /dashboard redirect, got %q", got)
		}
	})
}

func newDashboardServerForTests() *server.Server {
	return &server.Server{
		Env: &appenv.Env{
			AppName:   "miconsul",
			JWTSecret: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		},
	}
}
