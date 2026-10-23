package theme

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"miconsul/internal/database"
	"miconsul/internal/models"
	"miconsul/internal/server"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNewService(t *testing.T) {
	t.Run("nil server fails", func(t *testing.T) {
		_, err := NewService(nil)
		if err == nil {
			t.Fatalf("expected nil server error")
		}
	})

	t.Run("non-nil server succeeds", func(t *testing.T) {
		svc, err := NewService(&server.Server{})
		if err != nil {
			t.Fatalf("unexpected NewService error: %v", err)
		}
		if svc.Server == nil {
			t.Fatalf("expected service to keep server reference")
		}
	})
}

func TestHandleToggleTheme(t *testing.T) {
	t.Run("sets dark when checkbox checked", func(t *testing.T) {
		app := fiber.New()
		svc := service{Server: &server.Server{}}
		app.Post("/theme/toggle", svc.HandleToggleTheme)

		req := httptest.NewRequest(http.MethodPost, "/theme/toggle", strings.NewReader("theme=on&checked=dark&unchecked=light"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		setCookie := resp.Header.Get("Set-Cookie")
		if setCookie == "" {
			t.Fatalf("expected Set-Cookie header")
		}
		if !strings.Contains(setCookie, "theme=dark") {
			t.Fatalf("expected theme cookie set to dark, got %q", setCookie)
		}
	})

	t.Run("defaults to light when checkbox unchecked", func(t *testing.T) {
		app := fiber.New()
		svc := service{Server: &server.Server{}}
		app.Post("/theme/toggle", svc.HandleToggleTheme)

		req := httptest.NewRequest(http.MethodPost, "/theme/toggle", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		setCookie := resp.Header.Get("Set-Cookie")
		if setCookie == "" {
			t.Fatalf("expected Set-Cookie header")
		}
		if !strings.Contains(setCookie, "theme=light") {
			t.Fatalf("expected theme cookie set to light, got %q", setCookie)
		}
	})
}

func TestHandleToggleThemeSavesSignedInUserTheme(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:theme_toggle_user?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("migrate user: %v", err)
	}
	user := models.User{UID: "usr_theme", Email: "theme@example.com", Role: models.UserRoleUser, Theme: "light"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	svc := service{Server: &server.Server{DB: &database.Database{DB: db}}}
	app := fiber.New()
	app.Post("/theme/toggle", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleToggleTheme(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/theme/toggle", strings.NewReader("theme=on&checked=dark&unchecked=light"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	var saved models.User
	if err := db.First(&saved, user.ID).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	if saved.Theme != "dark" {
		t.Fatalf("expected saved theme dark, got %q", saved.Theme)
	}

	cookie := resp.Header.Get("Set-Cookie")
	if !strings.Contains(cookie, "theme=dark") {
		t.Fatalf("expected theme cookie set to dark, got %q", cookie)
	}
	for _, c := range resp.Cookies() {
		if c.Name == "theme" && time.Until(c.Expires) < 360*24*time.Hour {
			t.Fatalf("expected the theme cookie to last about a year, expires %v", c.Expires)
		}
	}
}

func TestHandleToggleThemeReadsTheToggleThatWasClicked(t *testing.T) {
	tests := []struct {
		body string
		want string
	}{
		{"theme=on&checked=dark&unchecked=light", "theme=dark"},
		{"checked=dark&unchecked=light", "theme=light"},
		{"theme=on&checked=light&unchecked=dark", "theme=light"},
		{"checked=light&unchecked=dark", "theme=dark"},
		{"theme=on&checked=blue&unchecked=light", "theme=light"},
		{"", "theme=light"},
	}

	for _, tt := range tests {
		app := fiber.New()
		svc := service{Server: &server.Server{}}
		app.Post("/theme/toggle", svc.HandleToggleTheme)

		req := httptest.NewRequest(http.MethodPost, "/theme/toggle", strings.NewReader(tt.body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("%q: request failed: %v", tt.body, err)
		}
		if got := resp.Header.Get("Set-Cookie"); !strings.Contains(got, tt.want) {
			t.Errorf("%q: expected cookie %s, got %q", tt.body, tt.want, got)
		}
	}
}

func TestHandleToggleThemeRefreshesOnlyWhenTheChoiceComesFromTheOS(t *testing.T) {
	tests := map[string]string{
		"theme=on&checked=light&unchecked=dark&from=os": "true",
		"theme=on&checked=dark&unchecked=light":         "",
	}

	for body, want := range tests {
		app := fiber.New()
		svc := service{Server: &server.Server{}}
		app.Post("/theme/toggle", svc.HandleToggleTheme)

		req := httptest.NewRequest(http.MethodPost, "/theme/toggle", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("%q: request failed: %v", body, err)
		}
		if got := resp.Header.Get("HX-Refresh"); got != want {
			t.Errorf("%q: HX-Refresh = %q, want %q", body, got, want)
		}
	}
}
