package user

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"miconsul/internal/model"
	"miconsul/internal/server"

	"github.com/gofiber/fiber/v3"
)

func TestNormalizeUserWriteInput(t *testing.T) {
	t.Run("nil user returns error", func(t *testing.T) {
		if err := normalizeUserWriteInput(nil); err == nil {
			t.Fatalf("expected nil user to fail")
		}
	})

	t.Run("trims and normalizes fields", func(t *testing.T) {
		u := &model.User{Name: "  User Name  ", Email: "  MIXED@Example.COM  ", Phone: "  +123  "}
		if err := normalizeUserWriteInput(u); err != nil {
			t.Fatalf("expected normalized input to pass: %v", err)
		}
		if u.Name != "User Name" || u.Email != "mixed@example.com" || u.Phone != "+123" {
			t.Fatalf("unexpected normalized user: %#v", u)
		}
	})

	t.Run("rejects over max lengths", func(t *testing.T) {
		cases := []model.User{
			{Name: strings.Repeat("n", 121)},
			{Email: strings.Repeat("e", 255)},
			{Phone: strings.Repeat("p", 41)},
		}
		for _, c := range cases {
			u := c
			if err := normalizeUserWriteInput(&u); err == nil {
				t.Fatalf("expected boundary validation error for %#v", c)
			}
		}
	})
}

func TestNewService(t *testing.T) {
	if _, err := NewService(nil); err == nil {
		t.Fatalf("expected nil server error")
	}

	svc, err := NewService(&server.Server{})
	if err != nil {
		t.Fatalf("unexpected NewService error: %v", err)
	}
	if svc.Server == nil {
		t.Fatalf("expected service to keep server reference")
	}
}

func TestUserProfileUpdateInputToUserProfileUpdates(t *testing.T) {
	in := userProfileUpdateInput{Name: "A", Email: "a@example.com", Phone: "1"}
	out := in.toUserProfileUpdates()
	if out.Name != in.Name || out.Email != in.Email || out.Phone != in.Phone {
		t.Fatalf("unexpected input mapping: %#v", out)
	}
}

func TestRespondWithRedirect(t *testing.T) {
	svc := service{Server: &server.Server{}}

	t.Run("non-htmx returns redirect", func(t *testing.T) {
		app := fiber.New()
		app.Get("/profile", func(c fiber.Ctx) error {
			return svc.respondWithRedirect(c, "/profile?toast=ok", fiber.StatusBadRequest)
		})

		req := httptest.NewRequest(http.MethodGet, "/profile", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); got != "/profile?toast=ok" {
			t.Fatalf("expected redirect location, got %q", got)
		}
	})

	t.Run("htmx sets HX-Location and status", func(t *testing.T) {
		app := fiber.New()
		app.Get("/profile", func(c fiber.Ctx) error {
			return svc.respondWithRedirect(c, "/profile?toast=ok", fiber.StatusBadRequest)
		})

		req := httptest.NewRequest(http.MethodGet, "/profile", nil)
		req.Header.Set("HX-Request", "true")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("expected %d, got %d", fiber.StatusBadRequest, resp.StatusCode)
		}
		if got := resp.Header.Get("HX-Location"); got != "/profile?toast=ok" {
			t.Fatalf("expected HX-Location header, got %q", got)
		}
	})
}
