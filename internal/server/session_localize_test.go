package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"miconsul/internal/lib/localize"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

func TestSessionReadWriteAndHelpers(t *testing.T) {
	s := &Server{SessionStore: session.NewStore()}
	app := fiber.New()

	app.Get("/rw", func(c fiber.Ctx) error {
		if err := s.SessionWrite(c, "key", "value"); err != nil {
			return err
		}
		if got := s.SessionRead(c, "key", "default"); got != "value" {
			t.Fatalf("expected session value, got %q", got)
		}
		if got := s.SessionRead(c, "missing", "default"); got != "default" {
			t.Fatalf("expected default value for missing key, got %q", got)
		}

		c.Locals("theme", "dark")
		if got := s.SessionUITheme(c); got != "dark" {
			t.Fatalf("expected dark theme, got %q", got)
		}

		return c.SendStatus(http.StatusNoContent)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/rw", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

func TestCurrentLocaleBranches(t *testing.T) {
	t.Run("without session store uses locals fallback", func(t *testing.T) {
		s := &Server{}
		app := fiber.New()
		app.Get("/locale", func(c fiber.Ctx) error {
			if got := s.CurrentLocale(c); got != "es-MX" {
				t.Fatalf("expected default locale es-MX, got %q", got)
			}
			c.Locals("locale", "en-US")
			if got := s.CurrentLocale(c); got != "en-US" {
				t.Fatalf("expected local fallback en-US, got %q", got)
			}
			return c.SendStatus(http.StatusNoContent)
		})

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/locale", nil))
		if err != nil || resp.StatusCode != http.StatusNoContent {
			t.Fatalf("locale request failed status=%d err=%v", resp.StatusCode, err)
		}
	})

	t.Run("with session store persists locale", func(t *testing.T) {
		s := &Server{SessionStore: session.NewStore()}
		app := fiber.New()
		app.Get("/locale", func(c fiber.Ctx) error {
			c.Locals("locale", "en-US")
			if got := s.CurrentLocale(c); got != "en-US" {
				t.Fatalf("expected en-US locale, got %q", got)
			}
			if got := s.CurrentLocale(c); got != "en-US" {
				t.Fatalf("expected persisted en-US locale, got %q", got)
			}
			return c.SendStatus(http.StatusNoContent)
		})

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/locale", nil))
		if err != nil || resp.StatusCode != http.StatusNoContent {
			t.Fatalf("locale request failed status=%d err=%v", resp.StatusCode, err)
		}
	})
}

func TestSessionIDTagAndLocalize(t *testing.T) {
	s := &Server{Localizer: localize.New("es-MX", "en-US")}
	app := fiber.New()

	app.Get("/meta", func(c fiber.Ctx) error {
		if got := s.L(c, "btn.save"); got == "" {
			t.Fatalf("expected localized value")
		}
		if got := s.SessionID(c); got != "sid123" {
			t.Fatalf("expected session id sid123, got %q", got)
		}
		if got := s.TagWithSessionID(c, "tag"); got != "sid123:tag" {
			t.Fatalf("expected tagged session id, got %q", got)
		}
		return c.SendStatus(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/meta", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sid123"})
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusNoContent {
		t.Fatalf("meta request failed status=%d err=%v", resp.StatusCode, err)
	}
}

func TestLocalizeNilLocalizer(t *testing.T) {
	s := &Server{}
	app := fiber.New()
	app.Get("/l", func(c fiber.Ctx) error {
		if got := s.L(c, "btn.save"); got != "" {
			t.Fatalf("expected empty translation when localizer is nil, got %q", got)
		}
		return c.SendStatus(http.StatusNoContent)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/l", nil))
	if err != nil || resp.StatusCode != http.StatusNoContent {
		t.Fatalf("localize nil request failed status=%d err=%v", resp.StatusCode, err)
	}
}
