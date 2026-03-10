package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

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

func TestAuthHandlersSignupPaths(t *testing.T) {
	t.Run("HandleSignupPage redirects logged-in user", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Get("/signup", func(c fiber.Ctx) error {
			c.Locals("current_user", model.User{ID: "user_1"})
			return svc.HandleSignupPage(c)
		})

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/signup", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); got != "/todos" {
			t.Fatalf("expected /todos redirect, got %q", got)
		}
	})

	t.Run("HandleSignup returns form errors for invalid input", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Post("/signup", svc.HandleSignup)

		form := url.Values{"email": {"bad-email"}, "password": {"Password1!"}, "confirm": {"Password1!"}}
		req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200 signup re-render, got %d", resp.StatusCode)
		}
	})

	t.Run("HandleSignup rejects password mismatch", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Post("/signup", svc.HandleSignup)

		form := url.Values{"email": {"new@example.com"}, "password": {"Password1!"}, "confirm": {"Different1!"}}
		req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200 signup re-render, got %d", resp.StatusCode)
		}
	})

	t.Run("HandleSignup redirects to signin for pending confirmation", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUserWithOptions(t, svc, authUserSeedOptions{
			Email:        "pending2@example.com",
			Password:     "Password1!",
			ConfirmToken: "pending-token",
		})

		app := fiber.New()
		app.Post("/signup", svc.HandleSignup)

		form := url.Values{"email": {"pending2@example.com"}, "password": {"Password1!"}, "confirm": {"Password1!"}}
		req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
		if !strings.HasPrefix(resp.Header.Get("Location"), "/signin?") {
			t.Fatalf("expected redirect to signin, got %q", resp.Header.Get("Location"))
		}
	})

	t.Run("HandleSignup creates user and redirects to signin", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Post("/signup", svc.HandleSignup)

		form := url.Values{"email": {"brandnew@example.com"}, "password": {"Password1!"}, "confirm": {"Password1!"}}
		req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
		if !strings.HasPrefix(resp.Header.Get("Location"), "/signin?") {
			t.Fatalf("expected redirect to signin, got %q", resp.Header.Get("Location"))
		}
	})
}

func TestAuthHandlersSignupConfirmEmailPaths(t *testing.T) {
	t.Run("blank token redirects to signin", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Get("/signup/confirm", svc.HandleSignupConfirmEmail)

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/signup/confirm", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
	})

	t.Run("unknown token redirects to signin", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Get("/signup/confirm/:token", svc.HandleSignupConfirmEmail)

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/signup/confirm/missing", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
	})

	t.Run("valid token sets auth cookie and redirects", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUserWithOptions(t, svc, authUserSeedOptions{
			Email:                 "confirm@example.com",
			Password:              "Password1!",
			ConfirmToken:          "confirm-ok-token",
			ConfirmTokenExpiresAt: time.Now().Add(time.Hour),
		})

		app := fiber.New()
		app.Get("/signup/confirm/:token", svc.HandleSignupConfirmEmail)

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/signup/confirm/confirm-ok-token", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
		if !strings.Contains(resp.Header.Get("Set-Cookie"), "Auth=") {
			t.Fatalf("expected auth cookie on successful confirmation")
		}
	})
}

func TestAuthHandlersResetPasswordPaths(t *testing.T) {
	t.Run("HandleResetPasswordPage renders", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Get("/resetpassword", svc.HandleResetPasswordPage)

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/resetpassword", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("HandleResetPassword rejects unknown user", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Post("/resetpassword", svc.HandleResetPassword)

		form := url.Values{"email": {"missing@example.com"}}
		req := httptest.NewRequest(http.MethodPost, "/resetpassword", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200 with form error, got %d", resp.StatusCode)
		}
	})

	t.Run("HandleResetPassword creates token for existing user", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUserWithOptions(t, svc, authUserSeedOptions{
			Email:    "reset@example.com",
			Password: "Password1!",
		})

		app := fiber.New()
		app.Post("/resetpassword", svc.HandleResetPassword)

		form := url.Values{"email": {"reset@example.com"}}
		req := httptest.NewRequest(http.MethodPost, "/resetpassword", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200 success render, got %d", resp.StatusCode)
		}
	})

	t.Run("HandleResetPasswordChange redirects on invalid token", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Get("/resetpassword/change/:token", svc.HandleResetPasswordChange)

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/resetpassword/change/invalid", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
	})

	t.Run("HandleResetPasswordUpdate validates token and nonce branches", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUserWithOptions(t, svc, authUserSeedOptions{
			Email:               "flow@example.com",
			Password:            "Password1!",
			ResetToken:          "reset-ok-token",
			ResetTokenExpiresAt: time.Now().Add(time.Hour),
		})

		app := fiber.New()
		app.Post("/resetpassword/change", svc.HandleResetPasswordUpdate)

		missingToken := url.Values{"email": {"flow@example.com"}, "nonce": {"abc"}, "password": {"Password1!"}, "confirm": {"Password1!"}}
		req1 := httptest.NewRequest(http.MethodPost, "/resetpassword/change", strings.NewReader(missingToken.Encode()))
		req1.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp1, err := app.Test(req1)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp1.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303 for missing token, got %d", resp1.StatusCode)
		}

		badNonce := url.Values{"email": {"flow@example.com"}, "token": {"reset-ok-token"}, "nonce": {""}, "password": {"Password1!"}, "confirm": {"Password1!"}}
		req2 := httptest.NewRequest(http.MethodPost, "/resetpassword/change", strings.NewReader(badNonce.Encode()))
		req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp2, err := app.Test(req2)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp2.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303 for nonce issue, got %d", resp2.StatusCode)
		}
	})

	t.Run("HandleResetPasswordUpdate validates password confirmation", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUserWithOptions(t, svc, authUserSeedOptions{
			Email:               "flow2@example.com",
			Password:            "Password1!",
			ResetToken:          "reset-ok-token-2",
			ResetTokenExpiresAt: time.Now().Add(time.Hour),
		})

		app := fiber.New()
		app.Post("/resetpassword/change", svc.HandleResetPasswordUpdate)

		blankPwd := url.Values{"email": {"flow2@example.com"}, "token": {"reset-ok-token-2"}, "nonce": {"nonce-1"}, "password": {""}, "confirm": {""}}
		req1 := httptest.NewRequest(http.MethodPost, "/resetpassword/change", strings.NewReader(blankPwd.Encode()))
		req1.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp1, err := app.Test(req1)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp1.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200 form error for blank password, got %d", resp1.StatusCode)
		}

		mismatch := url.Values{"email": {"flow2@example.com"}, "token": {"reset-ok-token-2"}, "nonce": {"nonce-2"}, "password": {"Password1!"}, "confirm": {"Different1!"}}
		req2 := httptest.NewRequest(http.MethodPost, "/resetpassword/change", strings.NewReader(mismatch.Encode()))
		req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp2, err := app.Test(req2)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp2.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200 form error for mismatch, got %d", resp2.StatusCode)
		}
	})
}

func TestRespondWithRedirect(t *testing.T) {
	svc := newAuthServiceForTests(t)

	t.Run("redirect without message keeps path", func(t *testing.T) {
		app := fiber.New()
		app.Get("/x", func(c fiber.Ctx) error {
			return svc.respondWithRedirect(c, "/signin", "")
		})

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/x", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if got := resp.Header.Get("Location"); got != "/signin" {
			t.Fatalf("expected /signin location, got %q", got)
		}
	})

	t.Run("redirect appends encoded message", func(t *testing.T) {
		app := fiber.New()
		app.Get("/x", func(c fiber.Ctx) error {
			return svc.respondWithRedirect(c, "/signin?from=test", "email pending confirmation")
		})

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/x", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		loc := resp.Header.Get("Location")
		if !strings.Contains(loc, "from=test") || !strings.Contains(loc, "msg=email+pending+confirmation") {
			t.Fatalf("expected encoded msg query in location, got %q", loc)
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

	for _, stmt := range []string{
		`ALTER TABLE users ADD COLUMN ConfirmEmailToken TEXT`,
		`ALTER TABLE users ADD COLUMN ConfirmEmailExpiresAt DATETIME`,
		`ALTER TABLE users ADD COLUMN ResetToken TEXT`,
		`ALTER TABLE users ADD COLUMN ResetTokenExpiresAt DATETIME`,
	} {
		_ = gdb.Exec(stmt).Error
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

	return seedAuthUserWithOptions(t, svc, authUserSeedOptions{
		Email:        email,
		Password:     password,
		ConfirmToken: confirmToken,
	})
}

type authUserSeedOptions struct {
	Email                 string
	Password              string
	ConfirmToken          string
	ConfirmTokenExpiresAt time.Time
	ResetToken            string
	ResetTokenExpiresAt   time.Time
}

func seedAuthUserWithOptions(t *testing.T, svc *service, opts authUserSeedOptions) model.User {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(opts.Password), 12)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	confirmExp := opts.ConfirmTokenExpiresAt
	if opts.ConfirmToken != "" && confirmExp.IsZero() {
		confirmExp = time.Now().Add(time.Hour)
	}

	u := model.User{
		Email:                 opts.Email,
		Password:              string(hash),
		ConfirmEmailToken:     opts.ConfirmToken,
		ConfirmEmailExpiresAt: confirmExp,
		ResetToken:            opts.ResetToken,
		ResetTokenExpiresAt:   opts.ResetTokenExpiresAt,
		Role:                  model.UserRoleUser,
	}
	if err := gorm.G[model.User](svc.DB.GormDB()).Create(context.Background(), &u); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	return u
}
