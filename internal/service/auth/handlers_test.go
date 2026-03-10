package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"
	"miconsul/internal/model"
	"miconsul/internal/server"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAuthHandlersSigninAndAPI(t *testing.T) {
	t.Run("HandleSigninPage redirects logged in users", func(t *testing.T) {
		svc := newAuthServiceForTests(t)

		app := fiber.New()
		app.Get("/signin", func(c fiber.Ctx) error {
			c.Locals("current_user", model.User{ID: "user_1"})
			return svc.HandleSigninPage(c)
		})

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/signin", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); got != "/" {
			t.Fatalf("expected redirect to /, got %q", got)
		}
	})

	t.Run("HandleSigninPage auto-redirects to logto when enabled", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		svc.Env.LogtoURL = "https://logto.example.com"
		svc.Env.LogtoAppID = "app-id"
		svc.Env.LogtoAppSecret = "secret"
		svc.Env.LogtoResource = "https://api.example.com"

		app := fiber.New()
		app.Get("/signin", svc.HandleSigninPage)

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/signin", nil))
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

	t.Run("HandleSignin returns login page on invalid credentials", func(t *testing.T) {
		svc := newAuthServiceForTests(t)

		app := fiber.New()
		app.Post("/signin", svc.HandleSignin)

		form := url.Values{"email": {"missing@example.com"}, "password": {"BadPassword1!"}}
		req := httptest.NewRequest(http.MethodPost, "/signin", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200 login re-render, got %d", resp.StatusCode)
		}
	})

	t.Run("HandleSignin sets cookie and redirects on success", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUser(t, svc, "ok@example.com", "Password1!", "")

		app := fiber.New()
		app.Post("/signin", svc.HandleSignin)

		form := url.Values{"email": {"ok@example.com"}, "password": {"Password1!"}}
		req := httptest.NewRequest(http.MethodPost, "/signin", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303 redirect, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); got != "/?timeframe=day" {
			t.Fatalf("expected redirect to dashboard, got %q", got)
		}
		if !strings.Contains(resp.Header.Get("Set-Cookie"), "Auth=") {
			t.Fatalf("expected auth cookie to be set")
		}
	})

	t.Run("HandleAPISignin validates bad JSON and blank fields", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Post("/api/auth/signin", svc.HandleAPISignin)

		badReq := httptest.NewRequest(http.MethodPost, "/api/auth/signin", strings.NewReader("{"))
		badReq.Header.Set("Content-Type", "application/json")
		badResp, err := app.Test(badReq)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if badResp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("expected 400 for invalid json, got %d", badResp.StatusCode)
		}

		blankReq := httptest.NewRequest(http.MethodPost, "/api/auth/signin", strings.NewReader("{}"))
		blankReq.Header.Set("Content-Type", "application/json")
		blankResp, err := app.Test(blankReq)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if blankResp.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("expected 400 for blank fields, got %d", blankResp.StatusCode)
		}
	})

	t.Run("HandleAPISignin returns 401 on invalid credentials", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Post("/api/auth/signin", svc.HandleAPISignin)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/signin", strings.NewReader(`{"email":"missing@example.com","password":"Password1!"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("HandleAPISignin returns 403 for pending confirmation", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUser(t, svc, "pending@example.com", "Password1!", "confirm-token")

		app := fiber.New()
		app.Post("/api/auth/signin", svc.HandleAPISignin)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/signin", strings.NewReader(`{"email":"pending@example.com","password":"Password1!"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.StatusCode)
		}
	})

	t.Run("HandleAPISignin returns 200 and cookie for valid credentials", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUser(t, svc, "api@example.com", "Password1!", "")

		app := fiber.New()
		app.Post("/api/auth/signin", svc.HandleAPISignin)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/signin", strings.NewReader(`{"email":"api@example.com","password":"Password1!","remember_me":true}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if !strings.Contains(resp.Header.Get("Set-Cookie"), "Auth=") {
			t.Fatalf("expected auth cookie to be set")
		}
	})
}

func TestAuthHandlersLogoutAndSessionEndpoints(t *testing.T) {
	t.Run("HandleLogout redirects for non-htmx", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.All("/logout", svc.HandleLogout)

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/logout", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); got != "/signin" {
			t.Fatalf("expected /signin redirect, got %q", got)
		}
	})

	t.Run("HandleLogout uses HX-Redirect for htmx", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.All("/logout", svc.HandleLogout)

		req := httptest.NewRequest(http.MethodGet, "/logout", nil)
		req.Header.Set("HX-Request", "true")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusTemporaryRedirect {
			t.Fatalf("expected 307, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("HX-Redirect"); got != "/signin" {
			t.Fatalf("expected HX-Redirect /signin, got %q", got)
		}
	})

	t.Run("HandleValidate returns 200", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Post("/api/auth/validate", svc.HandleValidate)

		resp, err := app.Test(httptest.NewRequest(http.MethodPost, "/api/auth/validate", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("HandleShowUser returns current user json", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Get("/api/auth/protected", func(c fiber.Ctx) error {
			c.Locals("current_user", model.User{ID: "user_123", Email: "u@example.com"})
			return svc.HandleShowUser(c)
		})

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/auth/protected", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
	})
}

func TestAuthHandlersResetPasswordShortCircuits(t *testing.T) {
	t.Run("HandleResetPasswordChange redirects when token is blank", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Get("/resetpassword/change", svc.HandleResetPasswordChange)

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/resetpassword/change", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
		if !strings.Contains(resp.Header.Get("Location"), "/resetpassword") {
			t.Fatalf("expected resetpassword redirect, got %q", resp.Header.Get("Location"))
		}
	})

	t.Run("HandleResetPasswordUpdate redirects when email is missing", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Post("/resetpassword/change", svc.HandleResetPasswordUpdate)

		req := httptest.NewRequest(http.MethodPost, "/resetpassword/change", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
		if !strings.Contains(resp.Header.Get("Location"), "something went wrong with the email") {
			t.Fatalf("expected error redirect message, got %q", resp.Header.Get("Location"))
		}
	})
}

func newAuthServiceForTests(t *testing.T) *service {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := gdb.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate user model: %v", err)
	}

	srv := &server.Server{
		Env: &appenv.Env{
			AppName:   "miconsul",
			JWTSecret: strings.Repeat("x", 32),
		},
		DB: &database.Database{DB: gdb},
	}

	svc, err := New(srv)
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}

	return svc
}

func seedAuthUser(t *testing.T, svc *service, email, password, confirmToken string) model.User {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	u := model.User{Email: email, Password: string(hash), ConfirmEmailToken: confirmToken, Role: model.UserRoleUser}
	if err := gorm.G[model.User](svc.DB.GormDB()).Create(context.Background(), &u); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	return u
}
