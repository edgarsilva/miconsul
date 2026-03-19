package clinic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"miconsul/internal/model"
	"miconsul/internal/server"

	"github.com/gofiber/fiber/v3"
)

func TestServiceIDGuards(t *testing.T) {
	svc := service{}
	ctx := context.Background()

	if _, err := svc.TakeClinicByID(ctx, "usr_1", ""); err != ErrIDRequired {
		t.Fatalf("expected ErrIDRequired from TakeClinicByID, got %v", err)
	}
	if err := svc.UpdateClinicByID(ctx, "usr_1", "", model.Clinic{}); err != ErrIDRequired {
		t.Fatalf("expected ErrIDRequired from UpdateClinicByID, got %v", err)
	}
	if err := svc.DeleteClinicByID(ctx, "usr_1", ""); err != ErrIDRequired {
		t.Fatalf("expected ErrIDRequired from DeleteClinicByID, got %v", err)
	}
	if _, err := svc.clinicExistsByID(ctx, "usr_1", ""); err != ErrIDRequired {
		t.Fatalf("expected ErrIDRequired from clinicExistsByID, got %v", err)
	}
}

func TestRespondHelpers(t *testing.T) {
	svc := &service{Server: &server.Server{}}

	t.Run("respondWithRedirect non-htmx", func(t *testing.T) {
		app := fiber.New()
		app.Get("/clinics", func(c fiber.Ctx) error {
			return svc.respondWithRedirect(c, "/clinics?toast=ok", fiber.StatusBadRequest)
		})

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/clinics", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
	})

	t.Run("respondWithClinicPage non-htmx", func(t *testing.T) {
		app := fiber.New()
		app.Get("/clinics", func(c fiber.Ctx) error {
			return svc.respondWithClinicPage(c, model.Clinic{ID: "cln_1"})
		})

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/clinics", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); got != "/clinics/cln_1" {
			t.Fatalf("expected clinic redirect, got %q", got)
		}
	})
}
