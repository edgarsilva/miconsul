package auth

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
)

func enableLogto(svc *service) {
	svc.Env.LogtoURL = "https://logto.example.com"
	svc.Env.LogtoAppID = "appid"
	svc.Env.LogtoAppSecret = "secret"
	svc.Env.LogtoResource = "https://api.example.com"
}

func authCookieSet(resp *http.Response) bool {
	for _, ck := range resp.Cookies() {
		if ck.Name == "Auth" && ck.Value != "" {
			return true
		}
	}
	return false
}

func TestLocalSigninWithLogtoEnabled(t *testing.T) {
	const pwd = "Password1!"

	tests := []struct {
		name       string
		role       models.UserRole
		password   string
		wantCookie bool
		wantStatus int
		wantMsg    string
	}{
		{"admin gets a local session", models.UserRoleAdmin, pwd, true, fiber.StatusOK, ""},
		{"regular user is refused", models.UserRoleUser, pwd, false, fiber.StatusForbidden, "admins only"},
		{"wrong password reveals no role", models.UserRoleUser, "wrong-password-1", false, fiber.StatusUnauthorized, "invalid credentials"},
	}

	for _, tt := range tests {
		t.Run(tt.name+" (form)", func(t *testing.T) {
			svc := newAuthServiceForTests(t)
			enableLogto(svc)
			user := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "fallback@example.com", Password: pwd, Role: tt.role})

			app := fiber.New()
			app.Post("/signin", svc.HandleSignin)

			form := url.Values{"email": {user.Email}, "password": {tt.password}}
			req := httptest.NewRequest(http.MethodPost, "/signin", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			if got := authCookieSet(resp); got != tt.wantCookie {
				t.Fatalf("Auth cookie set = %v, want %v", got, tt.wantCookie)
			}
			if !tt.wantCookie {
				body, _ := io.ReadAll(resp.Body)
				want := "incorrect email and password combination"
				if tt.wantStatus == fiber.StatusForbidden {
					want = "password sign-in is for admins only"
				}
				if !strings.Contains(string(body), want) {
					t.Fatalf("expected %q on the sign-in page", want)
				}
			}
		})

		t.Run(tt.name+" (api)", func(t *testing.T) {
			svc := newAuthServiceForTests(t)
			enableLogto(svc)
			user := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "fallback-api@example.com", Password: pwd, Role: tt.role})

			app := fiber.New()
			app.Post("/api/auth/signin", svc.HandleAPISignin)

			body := `{"email":"` + user.Email + `","password":"` + tt.password + `"}`
			req := httptest.NewRequest(http.MethodPost, "/api/auth/signin", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			if got := authCookieSet(resp); got != tt.wantCookie {
				t.Fatalf("Auth cookie set = %v, want %v", got, tt.wantCookie)
			}
			if tt.wantMsg != "" {
				var out map[string]string
				_ = json.NewDecoder(resp.Body).Decode(&out)
				if !strings.Contains(out["error"], tt.wantMsg) {
					t.Fatalf("error = %q, want it to mention %q", out["error"], tt.wantMsg)
				}
			}
		})
	}
}
