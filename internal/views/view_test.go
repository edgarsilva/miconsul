package views

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
)

type staticComponent struct{ body string }

func (c staticComponent) Render(_ context.Context, w io.Writer) error {
	_, err := w.Write([]byte(c.body))
	return err
}

func TestRender(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		return Render(c, staticComponent{body: "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("expected text/html content type, got %q", ct)
	}
}

func TestNewCtxAndOptions(t *testing.T) {
	runViewCtx(t, "/ctx?toast=Saved&sub=Done&level=success", func(c fiber.Ctx) {
		c.Locals("locale", "en-US")
		c.Locals("theme", "dark")
		expectedCU := models.User{UID: "usr_1", Email: "u@example.com"}
		c.Locals("current_user", expectedCU)

		vc, err := NewCtx(c)
		if err != nil {
			t.Fatalf("new ctx failed: %v", err)
		}
		if vc.Locale != "en-US" || vc.Theme != "dark" {
			t.Fatalf("unexpected locale/theme: %+v", vc)
		}
		if vc.CurrentUser.UID != expectedCU.UID {
			t.Fatalf("unexpected current user: %+v", vc.CurrentUser)
		}
		if vc.Toast.Msg != "Saved" || vc.Toast.Level != "success" {
			t.Fatalf("unexpected toast: %+v", vc.Toast)
		}
	})

	runViewCtx(t, "/ctx", func(c fiber.Ctx) {
		vc, err := NewCtx(c, WithTheme("dark"), WithLocale("en-US"), WithToast("ok", "sub", "info"), WithCurrentUser(models.User{UID: "usr_2"}))
		if err != nil {
			t.Fatalf("new ctx with options failed: %v", err)
		}
		if vc.Theme != "dark" || vc.Locale != "en-US" {
			t.Fatalf("unexpected option values: %+v", vc)
		}
		if vc.Toast.Msg != "ok" || vc.CurrentUser.UID != "usr_2" {
			t.Fatalf("unexpected option mapping: %+v", vc)
		}
	})
}

func TestWithThemeValidation(t *testing.T) {
	ctx := &Ctx{}
	if err := WithTheme("")(ctx); err == nil {
		t.Fatalf("expected blank theme error")
	}
	if err := WithTheme("blue")(ctx); err == nil {
		t.Fatalf("expected invalid theme error")
	}
	if err := WithTheme("light")(ctx); err != nil {
		t.Fatalf("expected valid theme, got %v", err)
	}
}

func TestQueryParams(t *testing.T) {
	runViewCtx(t, "/ctx?timeframe=week", func(c fiber.Ctx) {
		vc, err := NewCtx(c)
		if err != nil {
			t.Fatalf("new ctx failed: %v", err)
		}

		params := QueryParams(vc, "timeframe=day", "clinic=myclinic", "invalid")
		if !strings.Contains(params, "timeframe=day") {
			t.Fatalf("expected timeframe override, got %q", params)
		}
		if !strings.Contains(params, "clinic=myclinic") {
			t.Fatalf("expected clinic param, got %q", params)
		}
	})
}

func runViewCtx(t *testing.T, target string, fn func(c fiber.Ctx)) {
	t.Helper()

	app := fiber.New()
	app.Get("/ctx", func(c fiber.Ctx) error {
		fn(c)
		return c.SendStatus(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, target, nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected helper 204, got %d", resp.StatusCode)
	}
}
