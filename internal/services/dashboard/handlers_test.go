package dashboard

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestHandleAppointmentsSection(t *testing.T) {
	t.Run("renders only the section with dashboard filter links", func(t *testing.T) {
		svc, _ := newStatsServiceForTests(t)
		body := getAppointmentsSection(t, svc)

		if strings.Contains(body, "<html") {
			t.Errorf("section response includes the page layout")
		}
		for _, want := range []string{
			`id="appointments_section"`,
			`href="/dashboard?timeframe=week"`,
			`hx-get="/dashboard/appointments?timeframe=week"`,
			`hx-push-url="/dashboard?timeframe=week"`,
		} {
			if !strings.Contains(body, want) {
				t.Errorf("body missing %s", want)
			}
		}
		if strings.Contains(body, "hx-select") {
			t.Errorf("section narrows its response with hx-select")
		}
	})

	t.Run("adds an error toast when appointments fail to load", func(t *testing.T) {
		svc, _ := newStatsServiceForTests(t)
		if err := svc.DB.Migrator().DropTable(&models.Appointment{}); err != nil {
			t.Fatalf("drop appointments: %v", err)
		}
		body := getAppointmentsSection(t, svc)

		if !strings.Contains(body, `id="appointments_section"`) {
			t.Errorf("error response dropped the section")
		}
		if !strings.Contains(body, `<hx-partial hx-target="#toast_region"`) || !strings.Contains(body, "Failed to load appointments") {
			t.Errorf("body = %s, want an error toast partial", body)
		}
	})
}

func getAppointmentsSection(t *testing.T, svc service) string {
	t.Helper()

	app := fiber.New()
	app.Get("/dashboard/appointments", func(c fiber.Ctx) error {
		c.Locals("current_user", models.User{ID: 1, UID: "user_1"})
		return svc.HandleAppointmentsSection(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/dashboard/appointments?timeframe=day", nil)
	req.Header.Set("HX-Request", "true")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got status=%d err=%v", resp.StatusCode, err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	return string(body)
}

func newDashboardServerForTests() *server.Server {
	return &server.Server{
		Env: &appenv.Env{
			AppName:   "miconsul",
			JWTSecret: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		},
	}
}
