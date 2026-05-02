package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

func TestLogtoHandlersSessionFailureRedirects(t *testing.T) {
	t.Run("signin redirects when session load fails", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Get("/logto/signin", svc.HandleLogtoSignin)

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/logto/signin", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); got != "/signin?logto_error=session" {
			t.Fatalf("expected session error redirect, got %q", got)
		}
	})

	t.Run("callback redirects when session load fails", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Get("/logto/callback", svc.HandleLogtoCallback)

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/logto/callback", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); got != "/signin?logto_error=session" {
			t.Fatalf("expected session error redirect, got %q", got)
		}
	})

	t.Run("signout redirects to logto signin on session failure", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Get("/logto/signout", svc.HandleLogtoSignout)

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/logto/signout", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); got != "/logto/signin" {
			t.Fatalf("expected redirect to /logto/signin, got %q", got)
		}
	})

	t.Run("logto page redirects to signout on session failure", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Get("/logto", svc.HandleLogtoPage)

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/logto", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); got != "/logto/signout" {
			t.Fatalf("expected redirect to /logto/signout, got %q", got)
		}
	})
}

func TestLogtoHandlersCallbackErrorBranches(t *testing.T) {
	t.Run("callback redirects with callback error when signin state is missing", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		svc.SessionStore = session.NewStore()

		app := fiber.New()
		app.Get("/logto/callback", svc.HandleLogtoCallback)

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/logto/callback", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); got != "/signin?logto_error=callback" {
			t.Fatalf("expected callback error redirect, got %q", got)
		}
	})
}

func TestDeferLogtoSessionSave(t *testing.T) {
	deferLogtoSessionSave("logto route", func() error { return nil })
	deferLogtoSessionSave("logto route", func() error { return errors.New("save failed") })
}
