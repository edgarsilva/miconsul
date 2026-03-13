package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

func TestLocalStrategyAuthenticateFailures(t *testing.T) {
	t.Run("fails when token is missing", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		strategy := NewLocalStrategy(svc)

		app := fiber.New()
		app.Get("/", func(c fiber.Ctx) error {
			_, err := strategy.Authenticate(c)
			if err == nil {
				t.Fatalf("expected missing token error")
			}
			return c.SendStatus(fiber.StatusNoContent)
		})

		if _, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil)); err != nil {
			t.Fatalf("request failed: %v", err)
		}
	})

	t.Run("fails when token cannot be decoded", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		strategy := NewLocalStrategy(svc)

		app := fiber.New()
		app.Get("/", func(c fiber.Ctx) error {
			_, err := strategy.Authenticate(c)
			if err == nil {
				t.Fatalf("expected decode failure")
			}
			return c.SendStatus(fiber.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "Auth", Value: "not-a-jwt"})
		if _, err := app.Test(req); err != nil {
			t.Fatalf("request failed: %v", err)
		}
	})

	t.Run("fails when user does not exist", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		strategy := NewLocalStrategy(svc)

		token, err := JWTCreateTokenWithTTL(svc.Env, "missing@example.com", "user_missing", 2*time.Hour, false)
		if err != nil {
			t.Fatalf("create token: %v", err)
		}

		app := fiber.New()
		app.Get("/", func(c fiber.Ctx) error {
			_, err := strategy.Authenticate(c)
			if err == nil {
				t.Fatalf("expected missing user failure")
			}
			return c.SendStatus(fiber.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "Auth", Value: token})
		if _, err := app.Test(req); err != nil {
			t.Fatalf("request failed: %v", err)
		}
	})
}

func TestLocalStrategyAuthenticateSuccessAndRefresh(t *testing.T) {
	svc := newAuthServiceForTests(t)
	strategy := NewLocalStrategy(svc)
	user := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "local@example.com", Password: "Password1!"})

	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		authed, err := strategy.Authenticate(c)
		if err != nil {
			t.Fatalf("expected authenticate success, got %v", err)
		}
		if authed.ID != user.ID {
			t.Fatalf("expected authenticated user id %q, got %q", user.ID, authed.ID)
		}
		return c.SendStatus(fiber.StatusNoContent)
	})

	soonExpiring, err := JWTCreateTokenWithTTL(svc.Env, user.Email, user.ID, 30*time.Minute, false)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "Auth", Value: soonExpiring})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if !strings.Contains(resp.Header.Get("Set-Cookie"), "Auth=") {
		t.Fatalf("expected refreshed auth cookie")
	}
}

func TestRuntimeAuthenticateAndTakeUserByExtID(t *testing.T) {
	t.Run("Authenticate delegates to selected strategy", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		user := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "runtime@example.com", Password: "Password1!"})

		token, err := JWTCreateTokenWithTTL(svc.Env, user.Email, user.ID, time.Hour, false)
		if err != nil {
			t.Fatalf("create token: %v", err)
		}

		app := fiber.New()
		app.Get("/", func(c fiber.Ctx) error {
			authed, err := Authenticate(c, svc)
			if err != nil {
				t.Fatalf("expected runtime authenticate success, got %v", err)
			}
			if authed.ID != user.ID {
				t.Fatalf("expected %q, got %q", user.ID, authed.ID)
			}
			return c.SendStatus(fiber.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "Auth", Value: token})
		if _, err := app.Test(req); err != nil {
			t.Fatalf("request failed: %v", err)
		}
	})

	t.Run("TakeUserByExtID returns user when found", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		u := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "ext@example.com", Password: "Password1!"})
		u.ExtID = "logto_123"
		if err := svc.DB.Save(&u).Error; err != nil {
			t.Fatalf("update user ext id: %v", err)
		}

		got, err := TakeUserByExtID(context.Background(), svc, "logto_123")
		if err != nil {
			t.Fatalf("expected TakeUserByExtID success, got %v", err)
		}
		if got.ID != u.ID {
			t.Fatalf("expected user id %q, got %q", u.ID, got.ID)
		}
	})

	t.Run("TakeUserByExtID maps not found to auth error", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		_, err := TakeUserByExtID(context.Background(), svc, "missing")
		if err == nil || !strings.Contains(err.Error(), "failed to authenticate user") {
			t.Fatalf("expected mapped auth error, got %v", err)
		}
	})
}

func TestLogtoStorage(t *testing.T) {
	s := NewLogtoStorage(nil)
	if got := s.GetItem("missing"); got != "" {
		t.Fatalf("expected empty string for missing key, got %q", got)
	}
	s.SetItem("k", "v")
	if err := s.Save(); err != nil {
		t.Fatalf("expected save no-op success, got %v", err)
	}
}
