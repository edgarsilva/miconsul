package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	logtocore "github.com/logto-io/go/core"
)

func TestCredentialsFromRequest(t *testing.T) {
	t.Run("returns credentials from form", func(t *testing.T) {
		var gotEmail, gotPassword string
		var gotErr error

		runWithCtx(t, http.MethodPost, "/signin", "/signin", url.Values{
			"email":    {"user@example.com"},
			"password": {"s3cr3t"},
		}, func(c fiber.Ctx) {
			gotEmail, gotPassword, gotErr = credentialsFromRequest(c)
		})

		if gotErr != nil {
			t.Fatalf("expected no error, got %v", gotErr)
		}
		if gotEmail != "user@example.com" || gotPassword != "s3cr3t" {
			t.Fatalf("unexpected credentials: %q %q", gotEmail, gotPassword)
		}
	})

	t.Run("returns error when either field is blank", func(t *testing.T) {
		var gotErr error

		runWithCtx(t, http.MethodPost, "/signin", "/signin", url.Values{"email": {""}, "password": {""}}, func(c fiber.Ctx) {
			_, _, gotErr = credentialsFromRequest(c)
		})

		if gotErr == nil {
			t.Fatalf("expected error for blank credentials")
		}
	})
}

func TestResetPasswordEmailFromRequest(t *testing.T) {
	t.Run("reads email from form", func(t *testing.T) {
		var got string
		var gotErr error

		runWithCtx(t, http.MethodPost, "/reset", "/reset", url.Values{"email": {"form@example.com"}}, func(c fiber.Ctx) {
			got, gotErr = resetPasswordEmailFromRequest(c)
		})

		if gotErr != nil {
			t.Fatalf("expected no error, got %v", gotErr)
		}
		if got != "form@example.com" {
			t.Fatalf("expected form email, got %q", got)
		}
	})

	t.Run("reads email from params", func(t *testing.T) {
		var got string

		runWithCtx(t, http.MethodGet, "/reset/:email", "/reset/param@example.com", nil, func(c fiber.Ctx) {
			got, _ = resetPasswordEmailFromRequest(c)
		})

		if got != "param@example.com" {
			t.Fatalf("expected param email, got %q", got)
		}
	})

	t.Run("uses query when form and params are blank", func(t *testing.T) {
		var got string

		runWithCtx(t, http.MethodGet, "/reset", "/reset?email=query@example.com", nil, func(c fiber.Ctx) {
			got, _ = resetPasswordEmailFromRequest(c)
		})

		if got != "query@example.com" {
			t.Fatalf("expected query email, got %q", got)
		}
	})

	t.Run("returns error when email missing everywhere", func(t *testing.T) {
		var gotErr error

		runWithCtx(t, http.MethodGet, "/reset", "/reset", nil, func(c fiber.Ctx) {
			_, gotErr = resetPasswordEmailFromRequest(c)
		})

		if gotErr == nil {
			t.Fatalf("expected missing email error")
		}
	})
}

func TestTokenHelpers(t *testing.T) {
	tokens := map[string]func() string{
		"newSecretToken":        newSecretToken,
		"newConfirmEmailToken":  newConfirmEmailToken,
		"newResetPasswordToken": newResetPasswordToken,
	}
	for name, newToken := range tokens {
		token := newToken()
		if len(token) != 26 {
			t.Fatalf("%s: expected token length 26, got %d", name, len(token))
		}
		if strings.Trim(token, "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567") != "" {
			t.Fatalf("%s: expected base32 token, got %q", name, token)
		}
		if token == newToken() {
			t.Fatalf("%s: expected a new token per call", name)
		}
	}
}

func TestJWTHelpers(t *testing.T) {
	env := &appenv.Env{JWTSecret: strings.Repeat("x", 32)}
	testUser := models.User{Email: "u@example.com", UID: "uid_1"}

	t.Run("jwtSecretFromEnv validates config", func(t *testing.T) {
		if _, err := jwtSecretFromEnv(nil); err == nil {
			t.Fatalf("expected error for nil env")
		}
		if _, err := jwtSecretFromEnv(&appenv.Env{}); err == nil {
			t.Fatalf("expected error for empty secret")
		}
	})

	t.Run("token round-trips its claims", func(t *testing.T) {
		token, err := JWTCreateToken(env, NewAuthClaims(testUser, true))
		if err != nil {
			t.Fatalf("create token: %v", err)
		}

		claims, err := decodeJWTToken(env, token)
		if err != nil {
			t.Fatalf("decode token: %v", err)
		}
		if claims.Subject != testUser.Email || claims.UID != testUser.UID || !claims.RememberMe {
			t.Fatalf("unexpected claims: %+v", claims)
		}

		remaining := time.Until(claims.ExpiresAt.Time)
		if remaining < rememberAuthTokenTTL-time.Minute || remaining > rememberAuthTokenTTL {
			t.Fatalf("expected remaining close to remember-me ttl, got %v", remaining)
		}
	})

	t.Run("decodeJWTToken returns error on missing uid claim", func(t *testing.T) {
		secret, _ := jwtSecretFromEnv(env)
		token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{"sub": "u@example.com", "exp": time.Now().Add(time.Hour).Unix()})
		tokenStr, err := token.SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("sign token: %v", err)
		}

		if _, err := decodeJWTToken(env, tokenStr); err == nil {
			t.Fatalf("expected missing uid claim error")
		}
	})

	t.Run("decodeJWTToken reads tokens issued with map claims", func(t *testing.T) {
		secret, _ := jwtSecretFromEnv(env)
		token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
			"sub": "u@example.com",
			"uid": "uid_1",
			"rmb": true,
			"exp": time.Now().Add(time.Hour).Unix(),
		})
		tokenStr, err := token.SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("sign token: %v", err)
		}

		claims, err := decodeJWTToken(env, tokenStr)
		if err != nil {
			t.Fatalf("decode token: %v", err)
		}
		if claims.Subject != "u@example.com" || claims.UID != "uid_1" || !claims.RememberMe {
			t.Fatalf("unexpected claims: %+v", claims)
		}
	})

	t.Run("refresh keeps token if expiration is far", func(t *testing.T) {
		token := authTokenForTest(t, env, testUser, false, 2*time.Hour)

		claims, err := decodeJWTToken(env, token)
		if err != nil {
			t.Fatalf("decode token: %v", err)
		}

		refreshed, ttl, err := RefreshJWTToken(env, token, claims)
		if err != nil {
			t.Fatalf("refresh token: %v", err)
		}
		if refreshed != token {
			t.Fatalf("expected token unchanged when expiration is far")
		}
		if ttl <= time.Hour {
			t.Fatalf("expected ttl > 1h, got %v", ttl)
		}
	})

	t.Run("refresh issues new token when expiration is near", func(t *testing.T) {
		token := authTokenForTest(t, env, testUser, true, 30*time.Minute)

		claims, err := decodeJWTToken(env, token)
		if err != nil {
			t.Fatalf("decode token: %v", err)
		}

		refreshed, ttl, err := RefreshJWTToken(env, token, claims)
		if err != nil {
			t.Fatalf("refresh token: %v", err)
		}
		if refreshed == token {
			t.Fatalf("expected refreshed token to differ when expiration is near")
		}
		if ttl != rememberAuthTokenTTL {
			t.Fatalf("expected remember-me ttl %v, got %v", rememberAuthTokenTTL, ttl)
		}
	})

	t.Run("authTokenTTL helper", func(t *testing.T) {
		if authTokenTTL(true) != rememberAuthTokenTTL || authTokenTTL(false) != defaultAuthTokenTTL {
			t.Fatalf("unexpected token ttl helper values")
		}
	})
}

func TestLogtoStrategyHelpers(t *testing.T) {
	t.Run("logtoRedirectURI validates env and normalizes path", func(t *testing.T) {
		if _, err := logtoRedirectURI(nil, "/callback"); err == nil {
			t.Fatalf("expected error for nil env")
		}

		env := &appenv.Env{AppProtocol: "ftp", AppDomain: "example.com"}
		if _, err := logtoRedirectURI(env, "/callback"); err == nil {
			t.Fatalf("expected protocol validation error")
		}

		env = &appenv.Env{AppProtocol: "https", AppDomain: "bad/domain"}
		if _, err := logtoRedirectURI(env, "/callback"); err == nil {
			t.Fatalf("expected domain validation error")
		}

		env = &appenv.Env{AppProtocol: "HTTPS", AppDomain: "example.com"}
		uri, err := logtoRedirectURI(env, "callback")
		if err != nil {
			t.Fatalf("unexpected redirect uri error: %v", err)
		}
		if uri != "https://example.com/callback" {
			t.Fatalf("unexpected redirect uri: %q", uri)
		}
	})

	t.Run("logtoEnabled requires all critical fields", func(t *testing.T) {
		if logtoEnabled(nil) {
			t.Fatalf("expected disabled for nil env")
		}
		env := &appenv.Env{}
		if logtoEnabled(env) {
			t.Fatalf("expected disabled for empty env")
		}

		env.LogtoURL = "https://logto.example.com"
		env.LogtoAppID = "app-id-123456"
		env.LogtoAppSecret = "app-secret-123456"
		env.LogtoResource = "https://api.example.com"
		if !logtoEnabled(env) {
			t.Fatalf("expected enabled when all logto values present")
		}
	})

	t.Run("NewLogtoUser validates required claims", func(t *testing.T) {
		_, err := NewLogtoUser(logtocore.IdTokenClaims{})
		if err == nil {
			t.Fatalf("expected error for missing required claims")
		}

		claims := logtocore.IdTokenClaims{Sub: "sub_1", Email: "u@example.com", Name: "User"}
		user, err := NewLogtoUser(claims)
		if err != nil {
			t.Fatalf("unexpected NewLogtoUser error: %v", err)
		}
		if user.UID != "sub_1" || user.Sub != "sub_1" || user.Email != "u@example.com" {
			t.Fatalf("unexpected mapped logto user: %+v", user)
		}
	})

	t.Run("LogtoConfig maps env values", func(t *testing.T) {
		cfg := LogtoConfig(nil)
		if cfg == nil {
			t.Fatalf("expected non-nil default logto config")
		}

		env := &appenv.Env{
			LogtoURL:       "https://logto.example.com",
			LogtoAppID:     "appid",
			LogtoAppSecret: "secret",
			LogtoResource:  "https://api.example.com",
		}
		cfg = LogtoConfig(env)
		if cfg.Endpoint != env.LogtoURL || cfg.AppId != env.LogtoAppID || cfg.AppSecret != env.LogtoAppSecret {
			t.Fatalf("unexpected logto config mapping: %+v", cfg)
		}
		if len(cfg.Resources) != 1 || cfg.Resources[0] != env.LogtoResource {
			t.Fatalf("expected mapped logto resource, got %+v", cfg.Resources)
		}
	})
}

func runWithCtx(t *testing.T, method, routePath, target string, form url.Values, fn func(c fiber.Ctx)) {
	t.Helper()

	app := fiber.New()
	app.Add([]string{method}, routePath, func(c fiber.Ctx) error {
		fn(c)
		return c.SendStatus(http.StatusNoContent)
	})

	var bodyReader *strings.Reader
	if form != nil {
		bodyReader = strings.NewReader(form.Encode())
	} else {
		bodyReader = strings.NewReader("")
	}

	req := httptest.NewRequest(method, target, bodyReader)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("execute test request: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected status 204 from helper route, got %d", resp.StatusCode)
	}
}

func TestDecodeJWTTokenRejectsInvalidToken(t *testing.T) {
	env := &appenv.Env{JWTSecret: strings.Repeat("x", 32)}
	_, err := decodeJWTToken(env, "not-a-jwt")
	if err == nil {
		t.Fatalf("expected invalid token error")
	}
	if errors.Is(err, jwt.ErrTokenExpired) {
		t.Fatalf("expected parse error, not expiration error")
	}
}
