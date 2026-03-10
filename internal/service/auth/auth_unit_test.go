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
	t.Run("newHexToken returns expected hex length", func(t *testing.T) {
		token, err := newHexToken(16)
		if err != nil {
			t.Fatalf("newHexToken error: %v", err)
		}
		if len(token) != 32 {
			t.Fatalf("expected token length 32, got %d", len(token))
		}
		if strings.Trim(token, "0123456789abcdef") != "" {
			t.Fatalf("expected lowercase hex token, got %q", token)
		}
	})

	t.Run("newResetPasswordToken uses secure token size", func(t *testing.T) {
		token, err := newResetPasswordToken()
		if err != nil {
			t.Fatalf("newResetPasswordToken error: %v", err)
		}
		if len(token) != 64 {
			t.Fatalf("expected token length 64, got %d", len(token))
		}
	})

	t.Run("newConfirmEmailToken has non-empty token", func(t *testing.T) {
		token := newConfirmEmailToken()
		if token == "" {
			t.Fatalf("expected non-empty confirm email token")
		}
	})

	t.Run("randomTokenRunes honors charset and length", func(t *testing.T) {
		token := randomTokenRunes(24)
		if len(token) != 24 {
			t.Fatalf("expected token length 24, got %d", len(token))
		}
		if strings.Trim(token, "ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890") != "" {
			t.Fatalf("token contains invalid characters: %q", token)
		}
	})
}

func TestJWTHelpers(t *testing.T) {
	env := &appenv.Env{JWTSecret: strings.Repeat("x", 32)}

	t.Run("jwtSecretFromEnv validates config", func(t *testing.T) {
		if _, err := jwtSecretFromEnv(nil); err == nil {
			t.Fatalf("expected error for nil env")
		}
		if _, err := jwtSecretFromEnv(&appenv.Env{}); err == nil {
			t.Fatalf("expected error for empty secret")
		}
	})

	t.Run("JWTCreateTokenWithTTL defaults when ttl non-positive", func(t *testing.T) {
		token, err := JWTCreateTokenWithTTL(env, "u@example.com", "uid_1", 0, false)
		if err != nil {
			t.Fatalf("create token: %v", err)
		}

		claims, err := decodeJWTToken(env, token)
		if err != nil {
			t.Fatalf("decode token: %v", err)
		}

		exp, err := claims.GetExpirationTime()
		if err != nil {
			t.Fatalf("exp claim: %v", err)
		}

		remaining := time.Until(exp.Time)
		if remaining < defaultAuthTokenTTL-time.Minute || remaining > defaultAuthTokenTTL+time.Minute {
			t.Fatalf("expected remaining close to default ttl, got %v", remaining)
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

	t.Run("refresh keeps token if expiration is far", func(t *testing.T) {
		token, err := JWTCreateTokenWithTTL(env, "u@example.com", "uid_1", 2*time.Hour, false)
		if err != nil {
			t.Fatalf("create token: %v", err)
		}

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
		token, err := JWTCreateTokenWithTTL(env, "u@example.com", "uid_1", 30*time.Minute, true)
		if err != nil {
			t.Fatalf("create token: %v", err)
		}

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

	t.Run("rememberMeFromClaims and authTokenTTL helpers", func(t *testing.T) {
		if !rememberMeFromClaims(jwt.MapClaims{"rmb": true}) {
			t.Fatalf("expected rememberMe true")
		}
		if rememberMeFromClaims(jwt.MapClaims{"rmb": "true"}) {
			t.Fatalf("expected rememberMe false for non-bool claim")
		}
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
