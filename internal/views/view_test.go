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

		vc := NewCtx(c)
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
		vc := NewCtx(c, WithTheme("dark"), WithLocale("en-US"), WithToast("ok", "sub", "info"), WithCurrentUser(models.User{UID: "usr_2"}))
		if vc.Theme != "dark" || vc.Locale != "en-US" {
			t.Fatalf("unexpected option values: %+v", vc)
		}
		if vc.Toast.Msg != "ok" || vc.CurrentUser.UID != "usr_2" {
			t.Fatalf("unexpected option mapping: %+v", vc)
		}
	})
}

func TestNewCtxOptionsFallBackAndAllApply(t *testing.T) {
	runViewCtx(t, "/ctx", func(c fiber.Ctx) {
		c.Locals("theme", "dark")

		vc := NewCtx(c, WithTheme("blue"), WithLocale("en-US"), WithToast("Saved", "", "success"))
		if vc.Theme != "light" {
			t.Fatalf("invalid theme must fall back to light, got %q", vc.Theme)
		}
		if vc.Locale != "en-US" || vc.Toast.Msg != "Saved" || vc.Toast.Level != "success" {
			t.Fatalf("options after an invalid one must still apply, got locale=%q toast=%+v", vc.Locale, vc.Toast)
		}
	})
}

func TestWithThemeAndLocaleFallbacks(t *testing.T) {
	tests := []struct {
		name string
		opt  ContextOption
		want Ctx
	}{
		{"dark theme", WithTheme("dark"), Ctx{Theme: "dark"}},
		{"light theme", WithTheme("light"), Ctx{Theme: "light"}},
		{"blank theme", WithTheme(""), Ctx{Theme: "light"}},
		{"unknown theme", WithTheme("blue"), Ctx{Theme: "light"}},
		{"locale", WithLocale("en-US"), Ctx{Locale: "en-US"}},
		{"blank locale", WithLocale(""), Ctx{Locale: "es-MX"}},
	}

	for _, tt := range tests {
		var got Ctx
		tt.opt(&got)
		if got.Theme != tt.want.Theme || got.Locale != tt.want.Locale {
			t.Errorf("%s: got theme=%q locale=%q, want theme=%q locale=%q", tt.name, got.Theme, got.Locale, tt.want.Theme, tt.want.Locale)
		}
	}
}

func TestQueryParams(t *testing.T) {
	runViewCtx(t, "/ctx?timeframe=week", func(c fiber.Ctx) {
		vc := NewCtx(c)

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

func TestUserViewsOmitSecrets(t *testing.T) {
	user := models.User{
		UID:               "usr_1",
		Name:              "Ana Zyxwv",
		Email:             "ana@example.com",
		Role:              models.UserRoleAdmin,
		Password:          "hashed-password",
		ResetToken:        "reset-token-secret",
		ConfirmEmailToken: "confirm-token-secret",
	}

	runViewCtx(t, "/ctx", func(c fiber.Ctx) {
		vc := NewCtx(c, WithLocale("en-US"), WithCurrentUser(user))

		pages := map[string]interface {
			Render(context.Context, io.Writer) error
		}{
			"UsersIndexPage": UsersIndexPage(vc, []models.User{user}),
			"UserEditPage":   UserEditPage(vc, user),
		}
		for name, page := range pages {
			var html strings.Builder
			if err := page.Render(c.Context(), &html); err != nil {
				t.Fatalf("render %s: %v", name, err)
			}
			if !strings.Contains(html.String(), user.Name) {
				t.Fatalf("%s: expected the user's name in the page", name)
			}
			for _, secret := range []string{"hashed-password", "reset-token-secret", "confirm-token-secret"} {
				if strings.Contains(html.String(), secret) {
					t.Errorf("%s renders %q", name, secret)
				}
			}
		}
	})
}

func TestHTMLPageCSSURLStableAcrossRenders(t *testing.T) {
	runViewCtx(t, "/ctx", func(c fiber.Ctx) {
		vc := NewCtx(c)

		var first, second strings.Builder
		if err := HTMLPage(vc).Render(c.Context(), &first); err != nil {
			t.Fatalf("render first page: %v", err)
		}
		if err := HTMLPage(vc).Render(c.Context(), &second); err != nil {
			t.Fatalf("render second page: %v", err)
		}

		want := `href="/public/global.css?v=` + assetVersion + `"`
		for name, html := range map[string]string{"first": first.String(), "second": second.String()} {
			if !strings.Contains(html, want) {
				t.Errorf("%s render: expected %s in the page", name, want)
			}
		}
	})
}
