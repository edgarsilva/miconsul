package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
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

// fakeLogtoIDToken returns an unsigned ID token for sub; the Logto client
// only decodes the token stored in the session.
func fakeLogtoIDToken(sub string) string {
	enc := func(v string) string { return base64.RawURLEncoding.EncodeToString([]byte(v)) }
	now := time.Now().Unix()
	claims := fmt.Sprintf(`{"sub":%q,"iss":"https://logto.example.com/oidc","aud":"appid","iat":%d,"exp":%d}`, sub, now, now+3600)
	return enc(`{"alg":"RS256","typ":"JWT"}`) + "." + enc(claims) + "." + enc("sig")
}

func TestAuthenticateLocalAdminFallback(t *testing.T) {
	const pwd = "Password1!"

	tests := []struct {
		name         string
		logto        bool
		localRole    models.UserRole
		logtoSession bool
		wantUser     string
	}{
		{"logto on, admin local sign-in is accepted", true, models.UserRoleAdmin, false, "local"},
		{"logto on, regular local sign-in is rejected", true, models.UserRoleUser, false, ""},
		{"logto on, a logto session wins over a local admin", true, models.UserRoleAdmin, true, "logto"},
		{"logto off, local sign-in works as before", false, models.UserRoleUser, false, "local"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newAuthServiceForTests(t)
			svc.SessionStore = session.NewStore()
			if tt.logto {
				enableLogto(svc)
			}

			local := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "local@example.com", Password: pwd, Role: tt.localRole})
			logtoUser := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "logto@example.com", Password: pwd})
			if err := svc.DB.GormDB().Model(&logtoUser).Update("ext_id", "logto-sub-1").Error; err != nil {
				t.Fatalf("set ext id: %v", err)
			}
			token, err := JWTCreateToken(svc.Env, local.Email, local.UID)
			if err != nil {
				t.Fatalf("create token: %v", err)
			}

			app := fiber.New()
			app.Get("/seed", func(c fiber.Ctx) error {
				sess, err := svc.Session(c)
				if err != nil {
					return err
				}
				sess.Set("logto_id_token", fakeLogtoIDToken("logto-sub-1"))
				return sess.Save()
			})
			got := ""
			app.Get("/", func(c fiber.Ctx) error {
				user, err := Authenticate(c, svc)
				switch {
				case err != nil:
					got = ""
				case user.ID == local.ID:
					got = "local"
				case user.ID == logtoUser.ID:
					got = "logto"
				}
				return c.SendStatus(fiber.StatusNoContent)
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.AddCookie(&http.Cookie{Name: "Auth", Value: token})
			if tt.logtoSession {
				seed, err := app.Test(httptest.NewRequest(http.MethodGet, "/seed", nil), fiber.TestConfig{Timeout: 0})
				if err != nil {
					t.Fatalf("seed session: %v", err)
				}
				req.AddCookie(&http.Cookie{Name: "session_id", Value: cookieValueByName(seed, "session_id")})
			}
			if _, err := app.Test(req, fiber.TestConfig{Timeout: 0}); err != nil {
				t.Fatalf("request failed: %v", err)
			}
			if got != tt.wantUser {
				t.Fatalf("authenticated as %q, want %q", got, tt.wantUser)
			}
		})
	}
}

func TestLogoutRedirectWithLogtoEnabled(t *testing.T) {
	tests := []struct {
		name       string
		authCookie bool
		want       string
	}{
		{"local admin session goes back to the local form", true, "/signin?logged_out=1"},
		{"logto session signs out at logto", false, "/logto/signout"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newAuthServiceForTests(t)
			svc.SessionStore = session.NewStore()
			enableLogto(svc)

			app := fiber.New()
			app.Get("/logout", svc.HandleLogout)

			req := httptest.NewRequest(http.MethodGet, "/logout", nil)
			if tt.authCookie {
				req.AddCookie(&http.Cookie{Name: "Auth", Value: "some-jwt"})
			}
			resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			if got := resp.Header.Get("Location"); got != tt.want {
				t.Fatalf("redirect = %q, want %q", got, tt.want)
			}
		})
	}
}
