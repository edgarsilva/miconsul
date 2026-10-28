package auth

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"
	"miconsul/internal/models"
	"miconsul/internal/server"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/panjf2000/ants/v2"
	"go.opentelemetry.io/otel/trace/noop"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAuthHandlersSigninAndAPI(t *testing.T) {
	t.Run("HandleSigninPage redirects logged in users", func(t *testing.T) {
		svc := newAuthServiceForTests(t)

		app := fiber.New()
		app.Get("/signin", func(c fiber.Ctx) error {
			c.Locals("current_user", models.User{UID: "user_1"})
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

	t.Run("HandleSigninPage retries logto when provider reports error", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		svc.Env.LogtoURL = "https://logto.example.com"
		svc.Env.LogtoAppID = "app-id"
		svc.Env.LogtoAppSecret = "secret"
		svc.Env.LogtoResource = "https://api.example.com"

		app := fiber.New()
		app.Get("/signin", svc.HandleSigninPage)

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/signin?logto_error=denied", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303 redirect to retry logto, got %d", resp.StatusCode)
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
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200 login re-render, got %d", resp.StatusCode)
		}
	})

	t.Run("HandleSignin returns login page when credentials are missing", func(t *testing.T) {
		svc := newAuthServiceForTests(t)

		app := fiber.New()
		app.Post("/signin", svc.HandleSignin)

		req := httptest.NewRequest(http.MethodPost, "/signin", strings.NewReader("email=only@example.com"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200 login re-render, got %d", resp.StatusCode)
		}
	})

	t.Run("HandleSignin returns pending-confirmation message", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUserWithOptions(t, svc, authUserSeedOptions{
			Email:        "signinpending@example.com",
			Password:     "Password1!",
			ConfirmToken: "confirm-token",
		})

		app := fiber.New()
		app.Post("/signin", svc.HandleSignin)

		form := url.Values{"email": {"signinpending@example.com"}, "password": {"Password1!"}}
		req := httptest.NewRequest(http.MethodPost, "/signin", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200 login re-render, got %d", resp.StatusCode)
		}
	})

	t.Run("HandleSignin starts a session and redirects on success", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUser(t, svc, "ok@example.com", "Password1!", "")

		app := fiber.New()
		app.Post("/signin", svc.HandleSignin)

		form := url.Values{"email": {"ok@example.com"}, "password": {"Password1!"}}
		req := httptest.NewRequest(http.MethodPost, "/signin", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303 redirect, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); got != "/?timeframe=day" {
			t.Fatalf("expected redirect to dashboard, got %q", got)
		}
		assertSessionOnlyCookies(t, resp)
	})

	t.Run("HandleSignin redirects when the session can't be created", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUser(t, svc, "jwtfail@example.com", "Password1!", "")
		svc.SessionStore = nil

		app := fiber.New()
		app.Post("/signin", svc.HandleSignin)

		form := url.Values{"email": {"jwtfail@example.com"}, "password": {"Password1!"}}
		req := httptest.NewRequest(http.MethodPost, "/signin", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303 redirect on session failure, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); !strings.HasPrefix(got, "/?") {
			t.Fatalf("expected redirect to root with msg, got %q", got)
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
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
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
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.StatusCode)
		}
	})

	t.Run("HandleAPISignin returns 200 and a session for valid credentials", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUser(t, svc, "api@example.com", "Password1!", "")

		app := fiber.New()
		app.Post("/api/auth/signin", svc.HandleAPISignin)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/signin", strings.NewReader(`{"email":"api@example.com","password":"Password1!","remember_me":true}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		assertSessionOnlyCookies(t, resp)
	})

	t.Run("HandleAPISignin returns 500 when the session can't be created", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUser(t, svc, "apijwtfail@example.com", "Password1!", "")
		svc.SessionStore = nil

		app := fiber.New()
		app.Post("/api/auth/signin", svc.HandleAPISignin)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/signin", strings.NewReader(`{"email":"apijwtfail@example.com","password":"Password1!"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.StatusCode)
		}
	})

}

func TestAuthHandlersLogoutAndSessionEndpoints(t *testing.T) {
	t.Run("HandleLogout ends the session for every copy of its cookie", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUser(t, svc, "logout@example.com", "Password1!", "")

		app := fiber.New()
		app.Post("/signin", svc.HandleSignin)
		app.All("/logout", svc.HandleLogout)
		app.Get("/me", func(c fiber.Ctx) error {
			if _, err := NewLocalStrategy(svc).Authenticate(c); err != nil {
				return c.SendStatus(fiber.StatusUnauthorized)
			}
			return c.SendStatus(fiber.StatusOK)
		})

		form := url.Values{"email": {"logout@example.com"}, "password": {"Password1!"}}
		signinReq := httptest.NewRequest(http.MethodPost, "/signin", strings.NewReader(form.Encode()))
		signinReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		copied := cookieValueByName(testRequest(t, app, signinReq), "session_id")
		if copied == "" {
			t.Fatalf("expected session_id after sign in")
		}

		withCookie := func(method, path string) *http.Request {
			req := httptest.NewRequest(method, path, nil)
			req.AddCookie(&http.Cookie{Name: "session_id", Value: copied})
			return req
		}
		if resp := testRequest(t, app, withCookie(http.MethodGet, "/me")); resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected signed in before logout, got %d", resp.StatusCode)
		}

		testRequest(t, app, withCookie(http.MethodGet, "/logout"))

		if resp := testRequest(t, app, withCookie(http.MethodGet, "/me")); resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("expected the copied cookie to be rejected after logout, got %d", resp.StatusCode)
		}
	})

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
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
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

	t.Run("HandleLogout redirects to logto signout when enabled", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		svc.Env.LogtoURL = "https://logto.example.com"
		svc.Env.LogtoAppID = "app-id"
		svc.Env.LogtoAppSecret = "secret"
		svc.Env.LogtoResource = "https://api.example.com"

		app := fiber.New()
		app.All("/logout", svc.HandleLogout)

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/logout", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); got != "/logto/signout" {
			t.Fatalf("expected /logto/signout redirect, got %q", got)
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

	t.Run("HandleShowUser returns only public user fields", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Get("/api/auth/protected", func(c fiber.Ctx) error {
			c.Locals("current_user", models.User{
				ID:                    42,
				UID:                   "user_123",
				Email:                 "u@example.com",
				Password:              "hashed-password",
				ResetToken:            "reset-token-secret",
				ResetTokenExpiresAt:   time.Now().UTC().Add(time.Hour),
				ConfirmEmailToken:     "confirm-token-secret",
				ConfirmEmailExpiresAt: time.Now().UTC().Add(time.Hour),
			})
			return svc.HandleShowUser(c)
		})

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/auth/protected", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		var payload struct{ User map[string]any }
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode body %s: %v", body, err)
		}
		if payload.User["UID"] != "user_123" || payload.User["Email"] != "u@example.com" {
			t.Fatalf("expected UID and Email in response, got %s", body)
		}
		for _, key := range []string{"ID", "Password", "ResetToken", "ResetTokenExpiresAt", "ConfirmEmailToken", "ConfirmEmailExpiresAt"} {
			if _, ok := payload.User[key]; ok {
				t.Errorf("response must not include %s: %s", key, body)
			}
		}
		for _, secret := range []string{"hashed-password", "reset-token-secret", "confirm-token-secret"} {
			if strings.Contains(string(body), secret) {
				t.Errorf("response leaks %q: %s", secret, body)
			}
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
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
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
			c.Locals("current_user", models.User{UID: "user_1"})
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

	t.Run("HandleSignupPage renders for anonymous user", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Get("/signup", svc.HandleSignupPage)

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/signup", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("HandleSignupPage renders message query", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Get("/signup", svc.HandleSignupPage)

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/signup?msg=hello", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("HandleSignup returns form errors for invalid input", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Post("/signup", svc.HandleSignup)

		form := url.Values{"email": {"bad-email"}, "password": {"Password1!"}, "confirm": {"Password1!"}}
		req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200 signup re-render, got %d", resp.StatusCode)
		}
	})

	t.Run("HandleSignup hides SMTP errors from the page", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		svc.Env.EmailSMTPURL = "127.0.0.1"
		app := fiber.New()
		app.Post("/signup", svc.HandleSignup)

		form := url.Values{"email": {"smtpfail@example.com"}, "password": {"Password1!"}, "confirm": {"Password1!"}}
		req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200 signup re-render, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		page := string(body)
		if !strings.Contains(page, "failed to send confirmation email, please try again") {
			t.Fatalf("expected generic email error on page, got %q", page)
		}
		if strings.Contains(page, "127.0.0.1") || strings.Contains(page, "dial tcp") {
			t.Fatalf("expected SMTP error details to stay off the page, got %q", page)
		}
	})

	t.Run("HandleSignup returns form error when credentials are missing", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Post("/signup", svc.HandleSignup)

		req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
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
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
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
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
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
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
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

	t.Run("valid token redirects to signin without signing in", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUserWithOptions(t, svc, authUserSeedOptions{
			Email:                 "confirm@example.com",
			Password:              "Password1!",
			ConfirmToken:          "confirm-ok-token",
			ConfirmTokenExpiresAt: time.Now().UTC().Add(time.Hour),
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
		if cookie := resp.Header.Get("Set-Cookie"); cookie != "" {
			t.Fatalf("expected no cookie on confirmation, got %q", cookie)
		}
	})

	t.Run("confirm token update error still redirects to signin", func(t *testing.T) {
		svc := newAuthServiceForTestsWithUserUpdateError(t)
		seedAuthUserWithOptions(t, svc, authUserSeedOptions{
			Email:                 "confirm-update-err@example.com",
			Password:              "Password1!",
			ConfirmToken:          "confirm-update-err",
			ConfirmTokenExpiresAt: time.Now().UTC().Add(time.Hour),
		})

		app := fiber.New()
		app.Get("/signup/confirm/:token", svc.HandleSignupConfirmEmail)

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/signup/confirm/confirm-update-err", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303 redirect, got %d", resp.StatusCode)
		}
		if cookie := resp.Header.Get("Set-Cookie"); cookie != "" {
			t.Fatalf("did not expect auth cookie when update fails, got %q", cookie)
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
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200 with form error, got %d", resp.StatusCode)
		}
	})

	t.Run("HandleResetPassword reports an email it couldn't queue", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		user := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "reset@example.com", Password: "Password1!"})
		pool, err := ants.NewPool(1)
		if err != nil {
			t.Fatalf("new pool: %v", err)
		}
		pool.Release()
		if err := server.WithWorkPool(pool)(svc.Server); err != nil {
			t.Fatalf("set work pool: %v", err)
		}

		app := fiber.New()
		app.Post("/resetpassword", svc.HandleResetPassword)

		form := url.Values{"email": {user.Email}}
		req := httptest.NewRequest(http.MethodPost, "/resetpassword", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if !strings.Contains(string(body), "could not send the reset password email") {
			t.Fatalf("expected the send error on the page")
		}
		if strings.Contains(string(body), "check your email") {
			t.Fatalf("expected no success message when the email wasn't queued")
		}
	})

	t.Run("HandleResetPassword validates blank email", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Post("/resetpassword", svc.HandleResetPassword)

		req := httptest.NewRequest(http.MethodPost, "/resetpassword", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200 with validation error, got %d", resp.StatusCode)
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
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
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

	t.Run("HandleResetPasswordChange renders page for valid token", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUserWithOptions(t, svc, authUserSeedOptions{
			Email:               "change@example.com",
			Password:            "Password1!",
			ResetToken:          "change-ok-token",
			ResetTokenExpiresAt: time.Now().UTC().Add(time.Hour),
		})

		app := fiber.New()
		app.Get("/resetpassword/change/:token", svc.HandleResetPasswordChange)

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/resetpassword/change/change-ok-token", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200 render, got %d", resp.StatusCode)
		}
	})

	t.Run("HandleResetPasswordUpdate redirects when token is missing", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUserWithOptions(t, svc, authUserSeedOptions{
			Email:               "flow@example.com",
			Password:            "Password1!",
			ResetToken:          "reset-ok-token",
			ResetTokenExpiresAt: time.Now().UTC().Add(time.Hour),
		})

		app := fiber.New()
		app.Post("/resetpassword/change", svc.HandleResetPasswordUpdate)

		missingToken := url.Values{"email": {"flow@example.com"}, "password": {"Password1!"}, "confirm": {"Password1!"}}
		req1 := httptest.NewRequest(http.MethodPost, "/resetpassword/change", strings.NewReader(missingToken.Encode()))
		req1.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp1, err := app.Test(req1)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp1.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303 for missing token, got %d", resp1.StatusCode)
		}
		if loc := resp1.Header.Get("Location"); !strings.Contains(loc, "something went wrong with the token") {
			t.Fatalf("expected missing token redirect, got %q", loc)
		}
	})

	t.Run("reset password change page and update round trip", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUserWithOptions(t, svc, authUserSeedOptions{
			Email:               "roundtrip@example.com",
			Password:            "Password1!",
			ResetToken:          "roundtrip-token",
			ResetTokenExpiresAt: time.Now().UTC().Add(time.Hour),
		})

		app := fiber.New()
		app.Get("/resetpassword/change/:token", svc.HandleResetPasswordChange)
		app.Post("/resetpassword/change", svc.HandleResetPasswordUpdate)

		pageResp, err := app.Test(httptest.NewRequest(http.MethodGet, "/resetpassword/change/roundtrip-token", nil))
		if err != nil {
			t.Fatalf("change page request failed: %v", err)
		}
		page, err := io.ReadAll(pageResp.Body)
		if err != nil {
			t.Fatalf("read change page: %v", err)
		}

		form := url.Values{"password": {"NewPassword1!"}, "confirm": {"NewPassword1!"}}
		for _, m := range regexp.MustCompile(`<input type="hidden" name="(\w+)" value="([^"]*)"`).FindAllSubmatch(page, -1) {
			form.Set(string(m[1]), string(m[2]))
		}
		if form.Get("token") != "roundtrip-token" || form.Get("email") != "roundtrip@example.com" {
			t.Fatalf("expected token and email hidden inputs in the change page, got %v", form)
		}

		req := httptest.NewRequest(http.MethodPost, "/resetpassword/change", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
		if err != nil {
			t.Fatalf("update request failed: %v", err)
		}
		if got := resp.Header.Get("Location"); got != "/signin" {
			t.Fatalf("expected /signin redirect, got %q", got)
		}
	})

	t.Run("HandleResetPasswordUpdate validates password confirmation", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUserWithOptions(t, svc, authUserSeedOptions{
			Email:               "flow2@example.com",
			Password:            "Password1!",
			ResetToken:          "reset-ok-token-2",
			ResetTokenExpiresAt: time.Now().UTC().Add(time.Hour),
		})

		app := fiber.New()
		app.Post("/resetpassword/change", svc.HandleResetPasswordUpdate)

		blankPwd := url.Values{"email": {"flow2@example.com"}, "token": {"reset-ok-token-2"}, "password": {""}, "confirm": {""}}
		req1 := httptest.NewRequest(http.MethodPost, "/resetpassword/change", strings.NewReader(blankPwd.Encode()))
		req1.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp1, err := app.Test(req1)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp1.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200 form error for blank password, got %d", resp1.StatusCode)
		}

		mismatch := url.Values{"email": {"flow2@example.com"}, "token": {"reset-ok-token-2"}, "password": {"Password1!"}, "confirm": {"Different1!"}}
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

	t.Run("HandleResetPasswordUpdate redirects when token verify fails", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := fiber.New()
		app.Post("/resetpassword/change", svc.HandleResetPasswordUpdate)

		form := url.Values{
			"email":    {"unknown@example.com"},
			"token":    {"missing-token"},
			"password": {"Password1!"},
			"confirm":  {"Password1!"},
		}
		req := httptest.NewRequest(http.MethodPost, "/resetpassword/change", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303 for expired token path, got %d", resp.StatusCode)
		}
		if loc := resp.Header.Get("Location"); !strings.Contains(loc, "token has expired") {
			t.Fatalf("expected expired token redirect, got %q", loc)
		}
	})

	t.Run("HandleResetPasswordUpdate redirects when update fails", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUserWithOptions(t, svc, authUserSeedOptions{
			Email:               "updatefail@example.com",
			Password:            "Password1!",
			ResetToken:          "update-ok-token",
			ResetTokenExpiresAt: time.Now().UTC().Add(time.Hour),
		})

		app := fiber.New()
		app.Post("/resetpassword/change", svc.HandleResetPasswordUpdate)

		form := url.Values{
			"email":    {"different@example.com"},
			"token":    {"update-ok-token"},
			"password": {"Password1!"},
			"confirm":  {"Password1!"},
		}
		req := httptest.NewRequest(http.MethodPost, "/resetpassword/change", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303 on update failure, got %d", resp.StatusCode)
		}
		if loc := resp.Header.Get("Location"); !strings.Contains(loc, "something went wrong, try again") {
			t.Fatalf("expected update failure redirect, got %q", loc)
		}
	})

	t.Run("HandleResetPasswordUpdate redirects to signin on success", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUserWithOptions(t, svc, authUserSeedOptions{
			Email:               "updatesuccess@example.com",
			Password:            "Password1!",
			ResetToken:          "update-success-token",
			ResetTokenExpiresAt: time.Now().UTC().Add(time.Hour),
		})

		app := fiber.New()
		app.Post("/resetpassword/change", svc.HandleResetPasswordUpdate)

		form := url.Values{
			"email":    {"updatesuccess@example.com"},
			"token":    {"update-success-token"},
			"password": {"NewPassword1!"},
			"confirm":  {"NewPassword1!"},
		}
		req := httptest.NewRequest(http.MethodPost, "/resetpassword/change", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusSeeOther {
			t.Fatalf("expected 303 on success, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); got != "/signin" {
			t.Fatalf("expected /signin redirect, got %q", got)
		}
	})

	t.Run("HandleResetPassword shows error when reset token update fails", func(t *testing.T) {
		svc := newAuthServiceForTestsNoLegacyColumns(t)
		seedAuthUserWithOptions(t, svc, authUserSeedOptions{
			Email:    "reset-update-err@example.com",
			Password: "Password1!",
		})

		app := fiber.New()
		app.Post("/resetpassword", svc.HandleResetPassword)

		form := url.Values{"email": {"reset-update-err@example.com"}}
		req := httptest.NewRequest(http.MethodPost, "/resetpassword", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200 with update error message, got %d", resp.StatusCode)
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

// assertSessionOnlyCookies fails unless resp sets a session_id cookie and no
// Auth cookie.
func assertSessionOnlyCookies(t *testing.T, resp *http.Response) {
	t.Helper()

	if cookieValueByName(resp, "session_id") == "" {
		t.Fatalf("expected a session_id cookie")
	}
	if cookieValueByName(resp, "Auth") != "" {
		t.Fatalf("expected no Auth cookie")
	}
}

func newAuthServiceForTests(t *testing.T) *service {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := gdb.AutoMigrate(&models.User{}); err != nil {
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
		DB:             &database.Database{DB: gdb},
		SessionStore:   session.NewStore(),
		TracerProvider: noop.NewTracerProvider(),
	}

	svc, err := New(srv)
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}

	return svc
}

func newAuthServiceForTestsNoLegacyColumns(t *testing.T) *service {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := gdb.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("migrate user model: %v", err)
	}

	srv := &server.Server{
		Env: &appenv.Env{
			AppName:   "miconsul",
			JWTSecret: strings.Repeat("x", 32),
		},
		DB:             &database.Database{DB: gdb},
		SessionStore:   session.NewStore(),
		TracerProvider: noop.NewTracerProvider(),
	}

	svc, err := New(srv)
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}

	return svc
}

func newAuthServiceForTestsWithUserUpdateError(t *testing.T) *service {
	t.Helper()
	svc := newAuthServiceForTests(t)
	if err := svc.DB.Callback().Update().Before("gorm:update").Register("test_force_user_update_error", func(db *gorm.DB) {
		if db.Statement != nil && db.Statement.Table == "users" {
			db.AddError(errors.New("forced user update error"))
		}
	}); err != nil {
		t.Fatalf("register update callback: %v", err)
	}
	return svc
}

func seedAuthUser(t *testing.T, svc *service, email, password, confirmToken string) models.User {
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
	Theme                 string
	Locale                string
}

func seedAuthUserWithOptions(t *testing.T, svc *service, opts authUserSeedOptions) models.User {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(opts.Password), 12)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	confirmExp := opts.ConfirmTokenExpiresAt
	if opts.ConfirmToken != "" && confirmExp.IsZero() {
		confirmExp = time.Now().Add(time.Hour)
	}

	u := models.User{
		Email:                 opts.Email,
		Password:              string(hash),
		ConfirmEmailToken:     opts.ConfirmToken,
		ConfirmEmailExpiresAt: confirmExp,
		ResetToken:            opts.ResetToken,
		ResetTokenExpiresAt:   opts.ResetTokenExpiresAt,
		Theme:                 opts.Theme,
		Locale:                opts.Locale,
		Role:                  models.UserRoleUser,
	}
	if err := gorm.G[models.User](svc.DB.GormDB()).Create(context.Background(), &u); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	return u
}
