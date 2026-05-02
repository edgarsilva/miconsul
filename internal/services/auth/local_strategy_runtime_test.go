package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

func TestLocalStrategyAuthenticateFailures(t *testing.T) {
	t.Run("fails when token is missing", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		strategy := NewLocalStrategy(svc)

		app := fiber.New()
		app.Get("/", func(c fiber.Ctx) error {
			_, err := strategy.Authenticate(c)
			if err == nil {
				t.Fatalf("expected missing token error")
			}
			return c.SendStatus(fiber.StatusNoContent)
		})

		if _, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil)); err != nil {
			t.Fatalf("request failed: %v", err)
		}
	})

	t.Run("fails when token cannot be decoded", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		strategy := NewLocalStrategy(svc)

		app := fiber.New()
		app.Get("/", func(c fiber.Ctx) error {
			_, err := strategy.Authenticate(c)
			if err == nil {
				t.Fatalf("expected decode failure")
			}
			return c.SendStatus(fiber.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "Auth", Value: "not-a-jwt"})
		if _, err := app.Test(req); err != nil {
			t.Fatalf("request failed: %v", err)
		}
	})

	t.Run("fails when user does not exist", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		strategy := NewLocalStrategy(svc)

		token, err := JWTCreateTokenWithTTL(svc.Env, "missing@example.com", "user_missing", 2*time.Hour, false)
		if err != nil {
			t.Fatalf("create token: %v", err)
		}

		app := fiber.New()
		app.Get("/", func(c fiber.Ctx) error {
			_, err := strategy.Authenticate(c)
			if err == nil {
				t.Fatalf("expected missing user failure")
			}
			return c.SendStatus(fiber.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "Auth", Value: token})
		if _, err := app.Test(req); err != nil {
			t.Fatalf("request failed: %v", err)
		}
	})
}

func TestLocalStrategyAuthenticateSuccessAndRefresh(t *testing.T) {
	svc := newAuthServiceForTests(t)
	strategy := NewLocalStrategy(svc)
	user := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "local@example.com", Password: "Password1!"})

	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		authed, err := strategy.Authenticate(c)
		if err != nil {
			t.Fatalf("expected authenticate success, got %v", err)
		}
		if authed.UID != user.UID {
			t.Fatalf("expected authenticated user uid %q, got %q", user.UID, authed.UID)
		}
		return c.SendStatus(fiber.StatusNoContent)
	})

	soonExpiring, err := JWTCreateTokenWithTTL(svc.Env, user.Email, user.UID, 30*time.Minute, false)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "Auth", Value: soonExpiring})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if !strings.Contains(resp.Header.Get("Set-Cookie"), "Auth=") {
		t.Fatalf("expected refreshed auth cookie")
	}
}

func TestLocalStrategyAuthenticateSessionHydration(t *testing.T) {
	svc := newAuthServiceForTests(t)
	svc.SessionStore = session.NewStore()
	strategy := NewLocalStrategy(svc)
	user := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "session-cache@example.com", Password: "Password1!"})

	token, err := JWTCreateTokenWithTTL(svc.Env, user.Email, user.UID, 2*time.Hour, false)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		authed, err := strategy.Authenticate(c)
		if err != nil {
			t.Fatalf("expected authenticate success, got %v", err)
		}
		if authed.UID != user.UID {
			t.Fatalf("expected authenticated user uid %q, got %q", user.UID, authed.UID)
		}
		return c.SendStatus(fiber.StatusNoContent)
	})

	firstReq := httptest.NewRequest(http.MethodGet, "/", nil)
	firstReq.AddCookie(&http.Cookie{Name: "Auth", Value: token})
	firstResp, err := app.Test(firstReq)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}

	sessionID := cookieValueByName(firstResp, "session_id")
	if sessionID == "" {
		t.Fatalf("expected session_id cookie after first authenticate")
	}

	// If auth falls back to JWT decode, this would fail.
	svc.Env.JWTSecret = strings.Repeat("y", 32)

	secondReq := httptest.NewRequest(http.MethodGet, "/", nil)
	secondReq.AddCookie(&http.Cookie{Name: "Auth", Value: token})
	secondReq.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
	if _, err := app.Test(secondReq); err != nil {
		t.Fatalf("second request failed: %v", err)
	}
}

func TestLocalStrategyAuthenticateSessionHydrationTokenMismatch(t *testing.T) {
	svc := newAuthServiceForTests(t)
	svc.SessionStore = session.NewStore()
	strategy := NewLocalStrategy(svc)
	user := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "session-mismatch@example.com", Password: "Password1!"})

	validToken, err := JWTCreateTokenWithTTL(svc.Env, user.Email, user.UID, 2*time.Hour, false)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		_, err := strategy.Authenticate(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
		}
		return c.SendStatus(fiber.StatusNoContent)
	})

	firstReq := httptest.NewRequest(http.MethodGet, "/", nil)
	firstReq.AddCookie(&http.Cookie{Name: "Auth", Value: validToken})
	firstResp, err := app.Test(firstReq)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}

	sessionID := cookieValueByName(firstResp, "session_id")
	if sessionID == "" {
		t.Fatalf("expected session_id cookie after first authenticate")
	}

	badToken := "not-a-jwt"
	secondReq := httptest.NewRequest(http.MethodGet, "/", nil)
	secondReq.AddCookie(&http.Cookie{Name: "Auth", Value: badToken})
	secondReq.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
	secondResp, err := app.Test(secondReq)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	if secondResp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected unauthorized on token mismatch path, got %d", secondResp.StatusCode)
	}
}

func TestRuntimeAuthenticateAndTakeUserByExtID(t *testing.T) {
	t.Run("Authenticate delegates to selected strategy", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		user := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "runtime@example.com", Password: "Password1!"})

		token, err := JWTCreateTokenWithTTL(svc.Env, user.Email, user.UID, time.Hour, false)
		if err != nil {
			t.Fatalf("create token: %v", err)
		}

		app := fiber.New()
		app.Get("/", func(c fiber.Ctx) error {
			authed, err := Authenticate(c, svc)
			if err != nil {
				t.Fatalf("expected runtime authenticate success, got %v", err)
			}
			if authed.UID != user.UID {
				t.Fatalf("expected %q, got %q", user.UID, authed.UID)
			}
			return c.SendStatus(fiber.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "Auth", Value: token})
		if _, err := app.Test(req); err != nil {
			t.Fatalf("request failed: %v", err)
		}
	})

	t.Run("TakeUserByExtID returns user when found", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		u := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "ext@example.com", Password: "Password1!"})
		u.ExtID = "logto_123"
		if err := svc.DB.Save(&u).Error; err != nil {
			t.Fatalf("update user ext id: %v", err)
		}

		got, err := TakeUserByExtID(context.Background(), svc, "logto_123")
		if err != nil {
			t.Fatalf("expected TakeUserByExtID success, got %v", err)
		}
		if got.UID != u.UID {
			t.Fatalf("expected user uid %q, got %q", u.UID, got.UID)
		}
	})

	t.Run("TakeUserByExtID maps not found to auth error", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		_, err := TakeUserByExtID(context.Background(), svc, "missing")
		if err == nil || !strings.Contains(err.Error(), "failed to authenticate user") {
			t.Fatalf("expected mapped auth error, got %v", err)
		}
	})
}

func TestDecodeAuthSnapshot(t *testing.T) {
	t.Run("roundtrip encode decode", func(t *testing.T) {
		in := AuthSnapshot{
			Token:          "jwt-token",
			UserID:         "user_123",
			UserEmail:      "u@example.com",
			UserRole:       models.UserRoleAdmin,
			UserName:       "Ada",
			UserProfilePic: "/avatar.png",
			CachedAtUnix:   time.Now().Unix(),
		}

		out, ok := DecodeAuthSnapshot(EncodeAuthSnapshot(in))
		if !ok {
			t.Fatalf("expected decode success")
		}
		if out.Token != in.Token || out.UserID != in.UserID || out.UserEmail != in.UserEmail {
			t.Fatalf("decoded auth mismatch: got %+v", out)
		}
	})

	t.Run("fails for malformed cached timestamp", func(t *testing.T) {
		_, ok := DecodeAuthSnapshot("t=a&uid=u1&cat=not-a-number")
		if ok {
			t.Fatalf("expected decode failure")
		}
	})

	t.Run("fails when required fields missing", func(t *testing.T) {
		_, ok := DecodeAuthSnapshot("uid=u1&cat=123")
		if ok {
			t.Fatalf("expected decode failure when token is missing")
		}

		_, ok = DecodeAuthSnapshot("t=a&cat=123")
		if ok {
			t.Fatalf("expected decode failure when user id is missing")
		}
	})
}

func TestTokenDigest(t *testing.T) {
	t.Run("returns deterministic sha256 hash", func(t *testing.T) {
		token := "header.payload.signature"
		d1 := tokenDigest(token)
		d2 := tokenDigest(token)
		if d1 == "" || d1 != d2 {
			t.Fatalf("expected deterministic non-empty digest, got %q and %q", d1, d2)
		}
		if d1 == token {
			t.Fatalf("expected digest to differ from raw token")
		}
	})

	t.Run("returns empty for empty token", func(t *testing.T) {
		if got := tokenDigest(""); got != "" {
			t.Fatalf("expected empty digest, got %q", got)
		}
	})
}

func TestAuthSnapshotTTLBoundary(t *testing.T) {
	now := time.Unix(time.Now().Unix(), 0)
	token := tokenDigest("header.payload.signature")

	t.Run("accepts snapshot at ttl boundary", func(t *testing.T) {
		s := AuthSnapshot{Token: token, UserID: "u1", CachedAtUnix: now.Add(-authSessionHydrationTTL).Unix()}
		if !s.isValidForToken(token, now) {
			t.Fatalf("expected snapshot valid at ttl boundary")
		}
	})

	t.Run("rejects snapshot older than ttl", func(t *testing.T) {
		s := AuthSnapshot{Token: token, UserID: "u1", CachedAtUnix: now.Add(-authSessionHydrationTTL - time.Second).Unix()}
		if s.isValidForToken(token, now) {
			t.Fatalf("expected snapshot invalid past ttl")
		}
	})
}

func cookieValueByName(resp *http.Response, name string) string {
	for _, c := range resp.Cookies() {
		if c.Name == name {
			return c.Value
		}
	}

	return ""
}

func TestLogtoStorage(t *testing.T) {
	s := NewLogtoStorage(nil)
	if got := s.GetItem("missing"); got != "" {
		t.Fatalf("expected empty string for missing key, got %q", got)
	}
	s.SetItem("k", "v")
	if err := s.Save(); err != nil {
		t.Fatalf("expected save no-op success, got %v", err)
	}
}
