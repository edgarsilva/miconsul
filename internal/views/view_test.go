package views

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"miconsul/internal/models"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v3"
)

type staticComponent struct{ body string }

func (c staticComponent) Render(_ context.Context, w io.Writer) error {
	_, err := w.Write([]byte(c.body))
	return err
}

func TestRender(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(fc fiber.Ctx) error {
		return Render(fc, staticComponent{body: "ok"})
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
	runViewCtx(t, "/ctx", func(fc fiber.Ctx) {
		fc.Locals("locale", "en-US")
		fc.Locals("theme", "dark")
		expectedCU := models.User{UID: "usr_1", Email: "u@example.com"}
		fc.Locals("current_user", expectedCU)

		c := NewCtx(fc)
		if c.Locale != "en-US" || c.Theme != "dark" {
			t.Fatalf("unexpected locale/theme: %+v", c)
		}
		if c.CurrentUser.UID != expectedCU.UID {
			t.Fatalf("unexpected current user: %+v", c.CurrentUser)
		}
	})

	runViewCtx(t, "/ctx", func(fc fiber.Ctx) {
		c := NewCtx(fc, WithTheme("dark"), WithLocale("en-US"), WithToast("ok", "sub", LevelInfo), WithCurrentUser(models.User{UID: "usr_2"}))
		if c.Theme != "dark" || c.Locale != "en-US" {
			t.Fatalf("unexpected option values: %+v", c)
		}
		if c.Toast.Msg != "ok" || c.CurrentUser.UID != "usr_2" {
			t.Fatalf("unexpected option mapping: %+v", c)
		}
	})
}

func TestNewCtxOptionsFallBackAndAllApply(t *testing.T) {
	runViewCtx(t, "/ctx", func(fc fiber.Ctx) {
		fc.Locals("theme", "dark")

		c := NewCtx(fc, WithTheme("blue"), WithLocale("en-US"), WithToast("Saved", "", LevelSuccess))
		if c.Theme != "" {
			t.Fatalf("invalid theme must follow the OS, got %q", c.Theme)
		}
		if c.Locale != "en-US" || c.Toast.Msg != "Saved" || c.Toast.Level != LevelSuccess {
			t.Fatalf("options after an invalid one must still apply, got locale=%q toast=%+v", c.Locale, c.Toast)
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
		{"blank theme", WithTheme(""), Ctx{Theme: ""}},
		{"unknown theme", WithTheme("blue"), Ctx{Theme: ""}},
		{"locale", WithLocale("en-US"), Ctx{Locale: "en-US"}},
		{"blank locale", WithLocale(""), Ctx{Locale: "es-MX"}},
		{"unknown locale", WithLocale("fr-CA"), Ctx{Locale: "es-MX"}},
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
	runViewCtx(t, "/ctx?timeframe=week", func(fc fiber.Ctx) {
		c := NewCtx(fc)

		params := QueryParams(c, "timeframe=day", "clinic=myclinic", "invalid")
		if !strings.Contains(params, "timeframe=day") {
			t.Fatalf("expected timeframe override, got %q", params)
		}
		if !strings.Contains(params, "clinic=myclinic") {
			t.Fatalf("expected clinic param, got %q", params)
		}
	})
}

func runViewCtx(t *testing.T, target string, fn func(fc fiber.Ctx)) {
	t.Helper()

	app := fiber.New()
	app.Get("/ctx", func(fc fiber.Ctx) error {
		fn(fc)
		return fc.SendStatus(http.StatusNoContent)
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

	runViewCtx(t, "/ctx", func(fc fiber.Ctx) {
		c := NewCtx(fc, WithLocale("en-US"), WithCurrentUser(user))

		pages := map[string]interface {
			Render(context.Context, io.Writer) error
		}{
			"UserIndexPage": UserIndexPage(c, []models.User{user}),
			"UserFormPage":  UserFormPage(c, user),
		}
		for name, page := range pages {
			var html strings.Builder
			if err := page.Render(fc.Context(), &html); err != nil {
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

func TestRootLayoutCSSURLStableAcrossRenders(t *testing.T) {
	runViewCtx(t, "/ctx", func(fc fiber.Ctx) {
		c := NewCtx(fc)

		var first, second strings.Builder
		if err := RootLayout(c).Render(fc.Context(), &first); err != nil {
			t.Fatalf("render first page: %v", err)
		}
		if err := RootLayout(c).Render(fc.Context(), &second); err != nil {
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

func TestRootLayoutDataTheme(t *testing.T) {
	tests := []struct {
		theme string
		want  string
	}{
		{"", `<html lang="es-MX">`},
		{"light", `<html lang="es-MX" data-theme="cupcake">`},
		{"dark", `<html lang="es-MX" data-theme="cupcake">`},
	}

	for _, tt := range tests {
		runViewCtx(t, "/ctx", func(fc fiber.Ctx) {
			c := NewCtx(fc, WithTheme(tt.theme))

			var html strings.Builder
			if err := RootLayout(c).Render(fc.Context(), &html); err != nil {
				t.Fatalf("render: %v", err)
			}
			if !strings.Contains(html.String(), tt.want) {
				t.Errorf("theme %q: expected %s in the page", tt.theme, tt.want)
			}
		})
	}
}

func TestCmpThemeBtn(t *testing.T) {
	tests := []struct {
		theme       string
		wantSwaps   int
		wantChecked bool
		wantClasses []string
		wantValues  []string
	}{
		{"", 2, false, []string{`class="swap swap-rotate dark:hidden"`, `class="swap swap-rotate hidden dark:inline-grid"`}, []string{`value="dracula"`, `value="cupcake"`}},
		{"light", 1, false, []string{`class="swap swap-rotate"`}, []string{`value="dracula"`}},
		{"dark", 1, true, []string{`class="swap swap-rotate"`}, []string{`value="dracula"`}},
	}

	for _, tt := range tests {
		var html strings.Builder
		if err := CmpThemeBtn(tt.theme).Render(context.Background(), &html); err != nil {
			t.Fatalf("render: %v", err)
		}
		got := html.String()
		if n := strings.Count(got, `class="theme-controller"`); n != tt.wantSwaps {
			t.Errorf("theme %q: %d toggles, want %d", tt.theme, n, tt.wantSwaps)
		}
		if strings.Contains(got, `checked="true"`) != tt.wantChecked {
			t.Errorf("theme %q: checked = %v, want %v", tt.theme, !tt.wantChecked, tt.wantChecked)
		}
		for _, want := range append(tt.wantClasses, tt.wantValues...) {
			if !strings.Contains(got, want) {
				t.Errorf("theme %q: expected %s", tt.theme, want)
			}
		}
		if fromOS := strings.Count(got, "&#34;from&#34;:&#34;os&#34;"); fromOS != map[string]int{"": 2}[tt.theme] {
			t.Errorf("theme %q: %d toggles post from=os, want %d", tt.theme, fromOS, map[string]int{"": 2}[tt.theme])
		}
		if strings.Contains(got, " x-") {
			t.Errorf("theme %q: toggle must not use Alpine, got %s", tt.theme, got)
		}
	}
}

func TestRootLayoutLang(t *testing.T) {
	runViewCtx(t, "/ctx", func(fc fiber.Ctx) {
		for _, locale := range []string{"es-MX", "en-US"} {
			var html strings.Builder
			if err := RootLayout(NewCtx(fc, WithLocale(locale))).Render(fc.Context(), &html); err != nil {
				t.Fatalf("render page: %v", err)
			}
			if want := `<html lang="` + locale + `"`; !strings.Contains(html.String(), want) {
				t.Errorf("expected %s in the page", want)
			}
		}
	})
}

func TestUserProfileLanguageSelect(t *testing.T) {
	tests := []struct {
		name       string
		ctxLocale  string
		userLocale string
		want       string
	}{
		{"saved locale is selected", "es-MX", "en-US", "en-US"},
		{"request locale is selected without a saved one", "en-US", "", "en-US"},
		{"default", "es-MX", "", "es-MX"},
	}

	runViewCtx(t, "/ctx", func(fc fiber.Ctx) {
		for _, tt := range tests {
			user := models.User{UID: "usr_1", Name: "Ana", Locale: tt.userLocale}
			var html strings.Builder
			if err := UserProfile(NewCtx(fc, WithLocale(tt.ctxLocale)), user).Render(fc.Context(), &html); err != nil {
				t.Fatalf("%s: render profile: %v", tt.name, err)
			}

			got := html.String()
			if !strings.Contains(got, `<select name="locale"`) {
				t.Fatalf("%s: expected the language select", tt.name)
			}
			label := map[string]string{"es-MX": ">Idioma<", "en-US": ">Language<"}[tt.ctxLocale]
			if !strings.Contains(got, label) {
				t.Errorf("%s: expected the %s label", tt.name, label)
			}
			if want := `<option value="` + tt.want + `" lang="` + tt.want + `" selected>`; !strings.Contains(got, want) {
				t.Errorf("%s: expected %s", tt.name, want)
			}
			if n := strings.Count(got, " selected>"); n != 1 {
				t.Errorf("%s: expected one selected option, got %d", tt.name, n)
			}
		}
	})
}

func TestAuthPagesUseThemeColors(t *testing.T) {
	runViewCtx(t, "/ctx", func(fc fiber.Ctx) {
		c := NewCtx(fc)
		pages := map[string]interface {
			Render(context.Context, io.Writer) error
		}{
			"AuthLoginPage":  AuthLoginPage(c, "", nil),
			"AuthSignupPage": AuthSignupPage(c, "", nil),
		}
		for name, page := range pages {
			var html strings.Builder
			if err := page.Render(fc.Context(), &html); err != nil {
				t.Fatalf("render %s: %v", name, err)
			}
			if strings.Contains(html.String(), "fill-[#24292F]") {
				t.Errorf("%s: GitHub icon has a fixed fill that disappears on dark themes", name)
			}
		}

		var login strings.Builder
		if err := AuthLoginPage(c, "", nil).Render(fc.Context(), &login); err != nil {
			t.Fatalf("render AuthLoginPage: %v", err)
		}
		if !strings.Contains(login.String(), `<a href="/resetpassword" class="link font-semibold`) {
			t.Errorf("AuthLoginPage: expected the forgot password link in the base text color")
		}
	})
}

func TestDashboardViewAllLinks(t *testing.T) {
	runViewCtx(t, "/ctx", func(fc fiber.Ctx) {
		var html strings.Builder
		if err := DashboardMonthTrends(NewCtx(fc), DashboardStats{}).Render(fc.Context(), &html); err != nil {
			t.Fatalf("render: %v", err)
		}

		got := html.String()
		if strings.Contains(got, `href="#"`) {
			t.Errorf("a View all link still points at #")
		}
		for _, want := range []string{
			`href="/patients"`,
			`href="/appointments?timeframe=month"`,
			`href="/revenue"`,
		} {
			if !strings.Contains(got, want) {
				t.Errorf("expected %s", want)
			}
		}
	})
}

func TestIsSelected(t *testing.T) {
	tests := []struct {
		name        string
		uid         string
		selectedUID string
		i           int
		want        bool
	}{
		{"first item without a selection", "a", "", 0, true},
		{"other item without a selection", "b", "", 1, false},
		{"selected item", "b", "b", 1, true},
		{"first item with another selected", "a", "b", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pickerIsSelected(tt.uid, tt.selectedUID, tt.i); got != tt.want {
				t.Fatalf("isSelected(%q, %q, %d) = %v, want %v", tt.uid, tt.selectedUID, tt.i, got, tt.want)
			}
		})
	}
}

func TestNoAlpine(t *testing.T) {
	runViewCtx(t, "/ctx", func(fc fiber.Ctx) {
		c := NewCtx(fc, WithToast("Saved", "", LevelSuccess))
		for name, component := range map[string]templ.Component{
			"page":             RootLayout(c),
			"toast":            CmpToast("Saved", "", LevelSuccess),
			"password":         CmpPasswordInput(""),
			"confirm password": CmpConfirmPasswordInput(""),
		} {
			var html strings.Builder
			if err := component.Render(fc.Context(), &html); err != nil {
				t.Fatalf("render %s: %v", name, err)
			}
			got := html.String()
			for _, alpine := range []string{"alpinejs", " x-data", " x-show", " x-init", " x-bind", "@click"} {
				if strings.Contains(got, alpine) {
					t.Errorf("%s still uses Alpine (%s)", name, alpine)
				}
			}
		}
	})
}

func TestToastAndPasswordToggleMarkup(t *testing.T) {
	runViewCtx(t, "/ctx", func(fc fiber.Ctx) {
		var toast, password strings.Builder
		if err := CmpToast("Saved", "", LevelSuccess).Render(fc.Context(), &toast); err != nil {
			t.Fatalf("render toast: %v", err)
		}
		if err := CmpPasswordInput("").Render(fc.Context(), &password); err != nil {
			t.Fatalf("render password: %v", err)
		}

		for _, want := range []string{`toast-card`, `has-checked:hidden`, `type="checkbox"`} {
			if !strings.Contains(toast.String(), want) {
				t.Errorf("toast: expected %s", want)
			}
		}
		if !strings.Contains(password.String(), `hx-on:click=`) {
			t.Errorf("password: expected an hx-on:click toggle")
		}
	})
}

func TestNewCtxToastComesFromTheFlash(t *testing.T) {
	app := fiber.New()
	app.Get("/redirect", func(fc fiber.Ctx) error {
		return fc.Redirect().With("toast", "Saved").With("level", "success").To("/ctx")
	})
	var got Toast
	app.Get("/ctx", func(fc fiber.Ctx) error {
		got = NewCtx(fc).Toast
		return nil
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/redirect", nil))
	if err != nil {
		t.Fatalf("redirect request: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/ctx?toast=Spoofed&level=error", nil)
	for _, cookie := range resp.Cookies() {
		req.AddCookie(cookie)
	}
	if _, err := app.Test(req); err != nil {
		t.Fatalf("ctx request: %v", err)
	}

	if got.Msg != "Saved" || got.Level != LevelSuccess {
		t.Errorf("toast = %+v, want the flash toast, not the query string", got)
	}
}

func TestCmpFlashAlert(t *testing.T) {
	render := func(t *testing.T, flash *http.Cookie) string {
		t.Helper()

		app := fiber.New()
		app.Get("/", func(fc fiber.Ctx) error {
			return Render(fc, CmpFlashAlert(NewCtx(fc)))
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if flash != nil {
			req.AddCookie(flash)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		return string(body)
	}

	t.Run("renders the flash alert with its level", func(t *testing.T) {
		app := fiber.New()
		app.Get("/", func(fc fiber.Ctx) error {
			return fc.Redirect().With("alert", "Email confirmed").With("alert_level", "success").To("/signin")
		})
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		var flash *http.Cookie
		for _, cookie := range resp.Cookies() {
			if cookie.Name == fiber.FlashCookieName {
				flash = cookie
			}
		}
		if flash == nil {
			t.Fatalf("expected a flash cookie")
		}

		body := render(t, flash)
		if !strings.Contains(body, `role="alert"`) || !strings.Contains(body, "alert-success") || !strings.Contains(body, "Email confirmed") {
			t.Fatalf("expected a success alert, got %s", body)
		}
	})

	t.Run("renders nothing without a flash alert", func(t *testing.T) {
		if body := render(t, nil); strings.TrimSpace(body) != "" {
			t.Fatalf("expected no alert, got %s", body)
		}
	})
}

func TestAppointmentStartEndSessionPostsComplete(t *testing.T) {
	runViewCtx(t, "/ctx", func(fc fiber.Ctx) {
		appointment := models.Appointment{UID: "apnt_1"}

		var html strings.Builder
		if err := AppointmentStartSection(NewCtx(fc), appointment).Render(fc.Context(), &html); err != nil {
			t.Fatalf("render: %v", err)
		}

		got := html.String()
		if !strings.Contains(got, `type="button" hx-post="/appointments/apnt_1/complete"`) {
			t.Errorf("expected End session to hx-post /complete without submitting the save-session form")
		}
		if strings.Contains(got, "formaction") {
			t.Errorf("expected no formaction: htmx 4 ignores it on a form with hx-post")
		}
	})
}

func TestLevelRoundTripsThroughItsName(t *testing.T) {
	for _, level := range []Level{LevelInfo, LevelSuccess, LevelWarning, LevelError} {
		if got := parseLevel(level.String()); got != level {
			t.Errorf("parseLevel(%q) = %v, want %v", level.String(), got, level)
		}
	}

	if got := (Level{}).String(); got != "info" {
		t.Errorf("zero Level = %q, want info", got)
	}
	for _, name := range []string{"", "warn", "danger"} {
		if got := parseLevel(name); got != LevelInfo {
			t.Errorf("parseLevel(%q) = %v, want LevelInfo", name, got)
		}
	}
}

func TestCmpSearchInputTriggersAtMinSearchTermLen(t *testing.T) {
	runViewCtx(t, "/ctx", func(fc fiber.Ctx) {
		var html strings.Builder
		if err := CmpSearchInput(NewCtx(fc), "Search", "/patients/search", "#patients").Render(fc.Context(), &html); err != nil {
			t.Fatalf("render search input: %v", err)
		}

		want := fmt.Sprintf("this.value.length &gt;= %d", models.MinSearchTermLen)
		if !strings.Contains(html.String(), want) {
			t.Errorf("hx-trigger should search at %d characters, got %s", models.MinSearchTermLen, html.String())
		}
	})
}

func TestDashboardRevenueCard(t *testing.T) {
	cases := []struct {
		name    string
		revenue DashboardRevenue
		want    []string
		notWant []string
	}{
		{
			name:    "up from last month",
			revenue: DashboardRevenue{Total: 2450000, Previous: 2000000},
			want:    []string{"$24.5k", `title="$24,500"`, "22.5%", "text-success"},
			notWant: []string{"text-error"},
		},
		{
			name:    "down from last month",
			revenue: DashboardRevenue{Total: 96800, Previous: 100000},
			want:    []string{"$968", "3.2%", "text-error"},
		},
		{
			name:    "same as last month shows no change",
			revenue: DashboardRevenue{Total: 100000, Previous: 100000},
			want:    []string{"$1,000"},
			notWant: []string{"%"},
		},
		{
			name:    "no revenue last month shows no change",
			revenue: DashboardRevenue{Total: 150050},
			want:    []string{"$1.5k", `title="$1,501"`},
			notWant: []string{"%"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runViewCtx(t, "/ctx", func(fc fiber.Ctx) {
				var html strings.Builder
				stats := DashboardStats{Revenue: tc.revenue}
				if err := DashboardMonthTrends(NewCtx(fc), stats).Render(fc.Context(), &html); err != nil {
					t.Fatalf("render: %v", err)
				}

				got := html.String()
				card := got[strings.Index(got, "/public/images/funds.png"):]
				for _, want := range tc.want {
					if !strings.Contains(card, want) {
						t.Errorf("expected the revenue card to show %q", want)
					}
				}
				for _, notWant := range tc.notWant {
					if strings.Contains(card, notWant) {
						t.Errorf("expected the revenue card not to show %q", notWant)
					}
				}
				if strings.Contains(got, "24500<") {
					t.Errorf("expected no hard-coded revenue")
				}
			})
		})
	}
}

func TestDashboardCountsAreCompact(t *testing.T) {
	runViewCtx(t, "/ctx", func(fc fiber.Ctx) {
		var html strings.Builder
		stats := DashboardStats{
			Patients:     DashboardStat{Total: 1234, Diff: 12},
			Appointments: DashboardStat{Total: 20100, Diff: 1500},
		}
		if err := DashboardMonthTrends(NewCtx(fc), stats).Render(fc.Context(), &html); err != nil {
			t.Fatalf("render: %v", err)
		}

		got := html.String()
		for _, want := range []string{`title="1,234">1.2k<`, `title="20,100">20.1k<`, "1.5k"} {
			if !strings.Contains(got, want) {
				t.Errorf("expected %s", want)
			}
		}
	})
}

func TestRevenueIndexPage(t *testing.T) {
	runViewCtx(t, "/ctx", func(fc fiber.Ctx) {
		october := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
		november := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
		report := models.RevenueReport{
			Months: []models.RevenueMonth{
				{Start: november, Total: 185000, Appointments: 1},
				{Start: october, Total: 1850000, Appointments: 15},
			},
			Selected: models.RevenueMonth{Start: october, Total: 1850000, Appointments: 15},
			Patients: []models.RevenuePatient{
				{UID: "ptnt_alma", Name: "Alma", Total: 1250000, Appointments: 10, LastAppointment: october.AddDate(0, 0, 30)},
				{UID: "ptnt_bruno", Name: "Bruno", Deleted: true, Total: 600000, Appointments: 5, LastAppointment: october.AddDate(0, 0, 9)},
			},
			PatientCount: 37,
			Clinics: []models.RevenueClinic{
				{Name: "North", Total: 1200000, Appointments: 10},
				{Name: "South", Deleted: true, Total: 650000, Appointments: 5},
			},
			MonthlyAverage: 2425000,
		}

		var html strings.Builder
		if err := RevenueIndexPage(NewCtx(fc), report).Render(fc.Context(), &html); err != nil {
			t.Fatalf("render: %v", err)
		}

		got := html.String()
		for _, want := range []string{
			`data-chart="columns"`,
			`data-chart="donut"`,
			`<script defer src="/public/js/charts.js?v=` + assetVersion + `">`,
			`href="/revenue?month=2026-10"`,
			"$18,500",
			"65%",
			"$24,250",
			`href="/patients/ptnt_alma"`,
			"$12,500",
			"68%",
			"31 Oct",
			"Mostrando 2 de 37",
			"Promedio mensual",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("expected %s", want)
			}
		}
		i := strings.Index(got, `data-chart="columns" data-chart-data="`)
		if i < 0 || !strings.Contains(got[i:i+400], "values&#34;:[18500,1850]") {
			t.Errorf("expected the column chart values oldest first, in pesos: %s", got[max(i, 0):max(i, 0)+400])
		}
		if strings.Contains(got, `href="/patients/ptnt_bruno"`) {
			t.Errorf("expected no link to a deleted patient, whose page is gone")
		}
		if !strings.Contains(got, "badge-ghost") {
			t.Errorf("expected deleted clinics to be marked")
		}
	})
}

func TestChartsLayoutLoadsApexChartsBeforeTheChartsScript(t *testing.T) {
	runViewCtx(t, "/ctx", func(fc fiber.Ctx) {
		var html strings.Builder
		if err := RevenueIndexPage(NewCtx(fc), models.RevenueReport{}).Render(fc.Context(), &html); err != nil {
			t.Fatalf("render: %v", err)
		}

		got := html.String()
		headEnd := strings.Index(got, "</head>")
		head := got[:headEnd]
		if !strings.Contains(head, `<script defer src="https://cdn.jsdelivr.net/npm/apexcharts@6.10.0/dist/apexcharts.min.js" integrity="sha384-`) {
			t.Errorf("expected <head> to load ApexCharts deferred, with SRI")
		}
		if !strings.Contains(head, `apexcharts.css" integrity="sha384-`) {
			t.Errorf("expected the ApexCharts stylesheet with SRI in <head>")
		}
		script := strings.Index(got, `<script defer src="/public/js/charts.js`)
		lastChart := strings.LastIndex(got, "data-chart=")
		if script < headEnd || script < lastChart || strings.Count(got, "charts.js") != 1 {
			t.Errorf("expected charts.js once, deferred, in the page after its charts")
		}
	})
}

// Links into chart pages must be full page loads: htmx drops the <head> of
// boosted pages, and ChartsLayout loads ApexCharts there.
func TestLinksToRevenueAreNotBoosted(t *testing.T) {
	runViewCtx(t, "/ctx", func(fc fiber.Ctx) {
		c := NewCtx(fc, WithCurrentUser(models.User{UID: "usr_1"}))
		october := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
		report := models.RevenueReport{
			Months:   []models.RevenueMonth{{Start: october}},
			Selected: models.RevenueMonth{Start: october},
		}

		var html strings.Builder
		if err := RevenueIndexPage(c, report).Render(fc.Context(), &html); err != nil {
			t.Fatalf("render: %v", err)
		}

		got := html.String()
		if n := strings.Count(got, `<a href="/revenue" hx-boost="false"`); n != 2 {
			t.Errorf("expected both navbar Revenue links unboosted, got %d", n)
		}
		months := got[strings.Index(got, `data-chart="columns"`):]
		if strings.Contains(months[:strings.Index(months, "</table>")], "hx-boost") {
			t.Errorf("expected the month links not to be boosted")
		}
	})
}

func TestNavbarLinksToTheDashboard(t *testing.T) {
	runViewCtx(t, "/ctx", func(fc fiber.Ctx) {
		var html strings.Builder
		c := NewCtx(fc, WithCurrentUser(models.User{UID: "usr_1"}))
		if err := CmpNavbar(c).Render(fc.Context(), &html); err != nil {
			t.Fatalf("render: %v", err)
		}

		if n := strings.Count(html.String(), `<a href="/dashboard" class="btn btn-ghost`); n != 2 {
			t.Errorf("expected a Dashboard link in the desktop navbar and the mobile menu, got %d", n)
		}
	})
}

func TestPatientListShowsTheLastDoneAppointment(t *testing.T) {
	runViewCtx(t, "/ctx", func(fc fiber.Ctx) {
		patients := []models.Patient{
			{ID: 1, UID: "ptnt_seen", Name: "Seen"},
			{ID: 2, UID: "ptnt_unseen", Name: "Unseen"},
		}
		lastAppointments := map[uint]time.Time{
			1: time.Date(2026, time.October, 20, 16, 30, 0, 0, time.UTC),
		}

		var html strings.Builder
		if err := PatientList(NewCtx(fc), patients, lastAppointments).Render(fc.Context(), &html); err != nil {
			t.Fatalf("render: %v", err)
		}

		rows := strings.Split(html.String(), "<li ")[1:]
		if len(rows) != 2 {
			t.Fatalf("expected 2 patient rows, got %d", len(rows))
		}
		if !strings.Contains(rows[0], "Tue 20/Oct/26 10:30 AM") || strings.Contains(rows[0], "—") {
			t.Errorf("expected the seen patient's last appointment in the clinic timezone")
		}
		if !strings.Contains(rows[1], "—") {
			t.Errorf("expected a dash for the patient without a done appointment")
		}
	})
}
