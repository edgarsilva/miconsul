package user

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
)

func TestUserHandlerBranches(t *testing.T) {
	svc, currentUser := newUserServiceForTests(t)

	app := fiber.New()
	app.Get("/admin/users/:id", func(c fiber.Ctx) error {
		return svc.HandleEditPage(c)
	})
	app.Post("/profile", func(c fiber.Ctx) error {
		c.Locals("current_user", currentUser)
		return svc.HandleUpdateProfile(c)
	})
	app.Get("/api/users", func(c fiber.Ctx) error {
		return svc.HandleAPIUsers(c)
	})

	resp1, err := app.Test(httptest.NewRequest(http.MethodGet, "/admin/users/%20", nil))
	if err != nil || resp1.StatusCode != fiber.StatusSeeOther {
		t.Fatalf("edit page blank id expected 303, got status=%d err=%v", resp1.StatusCode, err)
	}

	badJSON := httptest.NewRequest(http.MethodPost, "/profile", strings.NewReader("{"))
	badJSON.Header.Set("Content-Type", "application/json")
	badJSON.Header.Set("HX-Request", "true")
	resp2, err := app.Test(badJSON)
	if err != nil || resp2.StatusCode != fiber.StatusNoContent {
		t.Fatalf("profile invalid body expected 204 for htmx redirect, got status=%d err=%v", resp2.StatusCode, err)
	}
	if got := resp2.Header.Get("HX-Redirect"); got == "" {
		t.Fatalf("expected HX-Redirect header for htmx profile error")
	}

	resp3, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/users", nil))
	if err != nil || resp3.StatusCode != fiber.StatusOK {
		t.Fatalf("api users expected 200, got status=%d err=%v", resp3.StatusCode, err)
	}
}

func TestHandleEditPageSuccess(t *testing.T) {
	svc, currentUser := newUserServiceForTests(t)

	admin := models.User{Email: "admin@example.com", Password: "hash", Role: models.UserRoleAdmin}
	if err := svc.DB.GormDB().Create(&admin).Error; err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	app := fiber.New()
	app.Get("/admin/users/:id", func(c fiber.Ctx) error {
		return svc.HandleEditPage(c)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/admin/users/"+currentUser.UID, nil))
	if err != nil || resp.StatusCode != fiber.StatusOK {
		t.Fatalf("edit page existing user expected 200, got status=%d err=%v", resp.StatusCode, err)
	}
}

func TestHandleUserProfilePicImgSrc(t *testing.T) {
	svc, currentUser := newUserServiceForTests(t)

	filename := currentUser.UID + "_preview"
	picPath, err := ProfilePicPath(filename, svc.Env.AssetsDir)
	if err != nil {
		t.Fatalf("profile pic path: %v", err)
	}
	if err := os.WriteFile(picPath, []byte("img-bytes"), 0o644); err != nil {
		t.Fatalf("write preview file: %v", err)
	}

	app := fiber.New()
	app.Get("/profile/avatar/preview", func(c fiber.Ctx) error {
		c.Locals("current_user", currentUser)
		return svc.HandleUserProfilePicImgSrc(c)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/profile/avatar/preview", nil))
	if err != nil || resp.StatusCode != fiber.StatusOK {
		t.Fatalf("self preview pic expected 200, got status=%d err=%v", resp.StatusCode, err)
	}

	resp2, err := app.Test(httptest.NewRequest(http.MethodGet, "/profile/avatar", nil))
	if err != nil || resp2.StatusCode != fiber.StatusNotFound {
		t.Fatalf("missing avatar pic expected 404, got status=%d err=%v", resp2.StatusCode, err)
	}
}

func TestHandleAPIUsersOmitsSecrets(t *testing.T) {
	svc, currentUser := newUserServiceForTests(t)
	currentUser.ResetToken = "reset-token-secret"
	currentUser.ConfirmEmailToken = "confirm-token-secret"
	if err := svc.DB.GormDB().Save(&currentUser).Error; err != nil {
		t.Fatalf("save user tokens: %v", err)
	}

	app := fiber.New()
	app.Get("/api/users", svc.HandleAPIUsers)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/users", nil))
	if err != nil || resp.StatusCode != fiber.StatusOK {
		t.Fatalf("api users expected 200, got status=%d err=%v", resp.StatusCode, err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), currentUser.UID) {
		t.Fatalf("expected user %s in response: %s", currentUser.UID, body)
	}
	for _, leak := range []string{"hash", "reset-token-secret", "confirm-token-secret", `"ID"`} {
		if strings.Contains(string(body), leak) {
			t.Errorf("response leaks %q: %s", leak, body)
		}
	}
}

func TestHandleUpdateProfileLocale(t *testing.T) {
	tests := []struct {
		name          string
		current       string
		posted        string
		wantSaved     string
		wantCookie    bool
		wantRefreshed bool
	}{
		{"new locale saves the cookie and reloads", "es-MX", "en-US", "en-US", true, true},
		{"current locale is saved without a cookie", "es-MX", "es-MX", "es-MX", false, false},
		{"unknown locale is ignored", "es-MX", "fr-CA", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, currentUser := newUserServiceForTests(t)

			app := fiber.New()
			app.Post("/profile", func(c fiber.Ctx) error {
				c.Locals("current_user", currentUser)
				c.Locals("locale", tt.current)
				return svc.HandleUpdateProfile(c)
			})

			form := url.Values{"name": {"Seed User"}, "email": {currentUser.Email}, "phone": {"5551234567"}, "locale": {tt.posted}}
			req := httptest.NewRequest(http.MethodPost, "/profile", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			resp, err := app.Test(req)
			if err != nil || resp.StatusCode != fiber.StatusOK {
				t.Fatalf("profile update expected 200, got status=%d err=%v", resp.StatusCode, err)
			}

			gotCookie := false
			for _, ck := range resp.Cookies() {
				if ck.Name == "locale" && ck.Value == tt.posted {
					gotCookie = true
				}
			}
			if gotCookie != tt.wantCookie {
				t.Errorf("locale cookie set = %v, want %v", gotCookie, tt.wantCookie)
			}
			if got := resp.Header.Get("HX-Refresh") == "true"; got != tt.wantRefreshed {
				t.Errorf("HX-Refresh = %v, want %v", got, tt.wantRefreshed)
			}

			saved, err := svc.TakeUserByUID(context.Background(), currentUser.UID)
			if err != nil {
				t.Fatalf("load user: %v", err)
			}
			if saved.Locale != tt.wantSaved {
				t.Errorf("saved locale = %q, want %q", saved.Locale, tt.wantSaved)
			}
		})
	}
}

func TestHandleAdminUpdateUser(t *testing.T) {
	svc, target := newUserServiceForTests(t)

	admin := models.User{Email: "admin@example.com", Password: "hash", Name: "Admin", Phone: "5550000000", Role: models.UserRoleAdmin}
	if err := svc.DB.GormDB().Create(&admin).Error; err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	app := fiber.New()
	app.Patch("/admin/users/:id", func(c fiber.Ctx) error {
		c.Locals("current_user", admin)
		return svc.HandleAdminUpdateUser(c)
	})

	form := url.Values{
		"id":     {admin.UID},
		"name":   {"Updated Name"},
		"email":  {"Updated@Example.com"},
		"phone":  {"5559876543"},
		"locale": {"en-US"},
	}
	req := httptest.NewRequest(http.MethodPatch, "/admin/users/"+target.UID, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != fiber.StatusOK {
		t.Fatalf("admin update expected 200, got status=%d err=%v", resp.StatusCode, err)
	}

	for _, ck := range resp.Cookies() {
		if ck.Name == "locale" {
			t.Errorf("admin update set a locale cookie: %q", ck.Value)
		}
	}

	saved, err := svc.TakeUserByUID(context.Background(), target.UID)
	if err != nil {
		t.Fatalf("load target: %v", err)
	}
	if saved.Name != "Updated Name" || saved.Email != "updated@example.com" || saved.Phone != "5559876543" {
		t.Errorf("target = %q %q %q, want the posted name, email and phone", saved.Name, saved.Email, saved.Phone)
	}
	if saved.Locale != target.Locale {
		t.Errorf("target locale = %q, want unchanged %q", saved.Locale, target.Locale)
	}

	adminAfter, err := svc.TakeUserByUID(context.Background(), admin.UID)
	if err != nil {
		t.Fatalf("load admin: %v", err)
	}
	if adminAfter.Name != admin.Name || adminAfter.Email != admin.Email || adminAfter.Phone != admin.Phone {
		t.Errorf("admin changed to %q %q %q", adminAfter.Name, adminAfter.Email, adminAfter.Phone)
	}
}

func TestHandleEditPageFormTarget(t *testing.T) {
	svc, target := newUserServiceForTests(t)

	admin := models.User{Email: "admin@example.com", Password: "hash", Role: models.UserRoleAdmin}
	if err := svc.DB.GormDB().Create(&admin).Error; err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	tests := []struct {
		name          string
		uid           string
		wantPath      string
		wantOwnFields bool
	}{
		{"admin editing another user saves to the admin route", target.UID, "/admin/users/" + target.UID, false},
		{"admin editing themselves saves to their profile", admin.UID, "/profile", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/admin/users/:id", func(c fiber.Ctx) error {
				c.Locals("current_user", admin)
				return svc.HandleEditPage(c)
			})

			resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/admin/users/"+tt.uid, nil))
			if err != nil || resp.StatusCode != fiber.StatusOK {
				t.Fatalf("edit page expected 200, got status=%d err=%v", resp.StatusCode, err)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			html := string(body)

			if want := `hx-patch="` + tt.wantPath + `"`; !strings.Contains(html, want) {
				t.Errorf("form missing %s", want)
			}
			if got := strings.Contains(html, `name="locale"`); got != tt.wantOwnFields {
				t.Errorf("locale select shown = %v, want %v", got, tt.wantOwnFields)
			}
			if got := strings.Contains(html, `name="profilePic"`); got != tt.wantOwnFields {
				t.Errorf("picture upload shown = %v, want %v", got, tt.wantOwnFields)
			}
		})
	}
}

func TestHandleAdminRemoveProfilePic(t *testing.T) {
	svc, target := newUserServiceForTests(t)

	admin := models.User{Email: "admin@example.com", Password: "hash", ProfilePic: "admin.png", Role: models.UserRoleAdmin}
	if err := svc.DB.GormDB().Create(&admin).Error; err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	if err := svc.DB.GormDB().Model(&models.User{}).Where("uid = ?", target.UID).Update("profile_pic", "target.png").Error; err != nil {
		t.Fatalf("seed target picture: %v", err)
	}

	app := fiber.New()
	app.Patch("/admin/users/:id/removepic", func(c fiber.Ctx) error {
		c.Locals("current_user", admin)
		return svc.HandleAdminRemoveProfilePic(c)
	})

	req := httptest.NewRequest(http.MethodPatch, "/admin/users/"+target.UID+"/removepic", nil)
	req.Header.Set("HX-Request", "true")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != fiber.StatusOK {
		t.Fatalf("admin remove picture expected 200, got status=%d err=%v", resp.StatusCode, err)
	}

	saved, err := svc.TakeUserByUID(context.Background(), target.UID)
	if err != nil {
		t.Fatalf("load target: %v", err)
	}
	if saved.ProfilePic != "" {
		t.Errorf("target picture = %q, want removed", saved.ProfilePic)
	}

	adminAfter, err := svc.TakeUserByUID(context.Background(), admin.UID)
	if err != nil {
		t.Fatalf("load admin: %v", err)
	}
	if adminAfter.ProfilePic != "admin.png" {
		t.Errorf("admin picture = %q, want unchanged", adminAfter.ProfilePic)
	}
}
