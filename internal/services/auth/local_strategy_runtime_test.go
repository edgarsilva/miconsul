package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

func TestLocalStrategyAuthenticateFailures(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{"no bearer token and no session", ""},
		{"bearer token cannot be decoded", "Bearer not-a-jwt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newAuthServiceForTests(t)
			app := newLocalStrategyTestApp(t, svc, models.User{})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			if resp := testRequest(t, app, req); resp.StatusCode != fiber.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", resp.StatusCode)
			}
		})
	}

	t.Run("bearer token of a missing user", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		app := newLocalStrategyTestApp(t, svc, models.User{})

		token, err := JWTCreateToken(svc.Env, "missing@example.com", "user_missing")
		if err != nil {
			t.Fatalf("create token: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		if resp := testRequest(t, app, req); resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})
}

func TestLocalStrategyBearerTokenIsStateless(t *testing.T) {
	svc := newAuthServiceForTests(t)
	user := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "bearer@example.com", Password: "Password1!"})
	app := newLocalStrategyTestApp(t, svc, user)

	token, err := JWTCreateToken(svc.Env, user.Email, user.UID)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := testRequest(t, app, req)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if got := responseBody(t, resp); got != user.UID {
		t.Fatalf("expected user %q, got %q", user.UID, got)
	}
	if cookie := resp.Header.Get("Set-Cookie"); cookie != "" {
		t.Fatalf("expected no cookie for a bearer request, got %q", cookie)
	}
}

func TestLocalStrategySignIn(t *testing.T) {
	t.Run("starts the session under a new ID", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		user := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "rotate@example.com", Password: "Password1!"})
		app := newLocalStrategyTestApp(t, svc, user)

		anonymousID := cookieValueByName(testRequest(t, app, httptest.NewRequest(http.MethodPost, "/touch", nil)), "session_id")
		if anonymousID == "" {
			t.Fatalf("expected an anonymous session_id")
		}

		signinReq := httptest.NewRequest(http.MethodPost, "/signin", nil)
		signinReq.AddCookie(&http.Cookie{Name: "session_id", Value: anonymousID})
		signinResp := testRequest(t, app, signinReq)
		if n := countCookies(signinResp, "session_id"); n != 1 {
			t.Fatalf("expected exactly one session_id cookie on sign in, got %d", n)
		}
		signedInID := cookieValueByName(signinResp, "session_id")
		if signedInID == "" || signedInID == anonymousID {
			t.Fatalf("expected a new session_id on sign in, got %q (was %q)", signedInID, anonymousID)
		}

		stale, err := svc.SessionStore.Storage.Get(anonymousID)
		if err != nil || stale != nil {
			t.Fatalf("expected the pre-sign-in session to be deleted, got %q, %v", stale, err)
		}

		if resp := authenticateWithSession(t, app, signedInID); resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected the new session to authenticate, got %d", resp.StatusCode)
		}
		if resp := authenticateWithSession(t, app, anonymousID); resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("expected the old session to stay anonymous, got %d", resp.StatusCode)
		}
	})

	tests := []struct {
		name       string
		path       string
		lifetime   time.Duration
		rememberMe bool
	}{
		{"token lasts 24h without remember me", "/signin", defaultAuthTokenTTL, false},
		{"token lasts 7 days with remember me", "/signin?remember=1", rememberAuthTokenTTL, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newAuthServiceForTests(t)
			user := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "ttl@example.com", Password: "Password1!"})
			app := newLocalStrategyTestApp(t, svc, user)

			sessionID := cookieValueByName(testRequest(t, app, httptest.NewRequest(http.MethodPost, tt.path, nil)), "session_id")
			claims := sessionTokenClaimsForTest(t, svc, sessionID)

			exp, err := claims.GetExpirationTime()
			if err != nil {
				t.Fatalf("token exp: %v", err)
			}
			if got := time.Until(exp.Time); got < tt.lifetime-time.Minute || got > tt.lifetime {
				t.Fatalf("token expires in %v, want %v", got, tt.lifetime)
			}
			if got := rememberMeFromClaims(claims); got != tt.rememberMe {
				t.Fatalf("token remember me = %v, want %v", got, tt.rememberMe)
			}
		})
	}

	t.Run("fails without a session store", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		svc.SessionStore = nil
		app := newLocalStrategyTestApp(t, svc, models.User{UID: "user_1"})

		resp := testRequest(t, app, httptest.NewRequest(http.MethodPost, "/signin", nil))
		if resp.StatusCode != fiber.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.StatusCode)
		}
		if got := responseBody(t, resp); !strings.HasPrefix(got, errAuthSessionCreate.Error()) {
			t.Fatalf("expected a create session error, got %q", got)
		}
	})
}

func TestLocalStrategySessionToken(t *testing.T) {
	rejected := []struct {
		name  string
		token func(svc *service, user models.User) string
	}{
		{"expired", func(svc *service, user models.User) string {
			return signTokenForTest(t, svc.Env.JWTSecret, jwt.MapClaims{"sub": user.Email, "uid": user.UID, "exp": time.Now().Add(-time.Minute).Unix()})
		}},
		{"signed with another secret", func(svc *service, user models.User) string {
			return signTokenForTest(t, strings.Repeat("z", 32), jwt.MapClaims{"sub": user.Email, "uid": user.UID, "exp": time.Now().Add(time.Hour).Unix()})
		}},
		{"issued for another user", func(svc *service, user models.User) string {
			return signTokenForTest(t, svc.Env.JWTSecret, jwt.MapClaims{"sub": "other@example.com", "uid": "user_other", "exp": time.Now().Add(time.Hour).Unix()})
		}},
	}
	for _, tt := range rejected {
		t.Run("rejects a token "+tt.name+" and signs the session out", func(t *testing.T) {
			svc := newAuthServiceForTests(t)
			user := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "token@example.com", Password: "Password1!"})
			app := newLocalStrategyTestApp(t, svc, user)
			sessionID := signInForTest(t, app)

			editSnapshotForTest(t, svc, sessionID, func(s *AuthSnapshot) { s.Token = tt.token(svc, user) })

			if resp := authenticateWithSession(t, app, sessionID); resp.StatusCode != fiber.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", resp.StatusCode)
			}
			if raw := sessionValueForTest(t, svc, sessionID, authSessionKey); raw != "" {
				t.Fatalf("expected the signed-in user removed from the session, got %q", raw)
			}
		})
	}

	t.Run("keeps a token far from expiry", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		user := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "keep@example.com", Password: "Password1!"})
		app := newLocalStrategyTestApp(t, svc, user)
		sessionID := signInForTest(t, app)
		before := snapshotForTest(t, svc, sessionID).Token

		if resp := authenticateWithSession(t, app, sessionID); resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		if after := snapshotForTest(t, svc, sessionID).Token; after != before {
			t.Fatalf("expected the session token unchanged")
		}
	})

	t.Run("refreshes a token about to expire and keeps remember me", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		user := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "refresh@example.com", Password: "Password1!"})
		app := newLocalStrategyTestApp(t, svc, user)
		sessionID := signInForTest(t, app)

		expiring, err := JWTCreateTokenWithTTL(svc.Env, user.Email, user.UID, 30*time.Minute, true)
		if err != nil {
			t.Fatalf("create token: %v", err)
		}
		editSnapshotForTest(t, svc, sessionID, func(s *AuthSnapshot) { s.Token = expiring })

		if resp := authenticateWithSession(t, app, sessionID); resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		claims := sessionTokenClaimsForTest(t, svc, sessionID)
		exp, err := claims.GetExpirationTime()
		if err != nil {
			t.Fatalf("token exp: %v", err)
		}
		if got := time.Until(exp.Time); got < rememberAuthTokenTTL-time.Minute {
			t.Fatalf("expected a refreshed 7 day token, expires in %v", got)
		}
		if !rememberMeFromClaims(claims) {
			t.Fatalf("expected the refreshed token to keep remember me")
		}
	})
}

func TestLocalStrategySessionHydration(t *testing.T) {
	t.Run("uses the cached user while fresh", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		user := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "cached@example.com", Password: "Password1!", Theme: "dark", Locale: "en-US"})
		app := newLocalStrategyTestApp(t, svc, user)
		sessionID := signInForTest(t, app)

		if err := svc.DB.Model(&models.User{}).Where("id = ?", user.ID).Update("name", "Renamed").Error; err != nil {
			t.Fatalf("rename user: %v", err)
		}

		resp := authenticateWithSession(t, app, sessionID)
		if got := responseBody(t, resp); got != user.UID {
			t.Fatalf("expected cached user %q, got %q", user.UID, got)
		}
		if got := resp.Header.Get("X-User-Name"); got != user.Name {
			t.Fatalf("expected cached name %q, got %q", user.Name, got)
		}
	})

	t.Run("reloads a stale user from the DB", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		user := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "stale@example.com", Password: "Password1!"})
		app := newLocalStrategyTestApp(t, svc, user)
		sessionID := signInForTest(t, app)
		ageSnapshotForTest(t, svc, sessionID)

		if err := svc.DB.Model(&user).Update("name", "Renamed").Error; err != nil {
			t.Fatalf("rename user: %v", err)
		}

		if got := authenticateWithSession(t, app, sessionID).Header.Get("X-User-Name"); got != "Renamed" {
			t.Fatalf("expected reloaded name, got %q", got)
		}
	})

	t.Run("rejects sessions signed in before the session held a JWT", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		user := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "legacy@example.com", Password: "Password1!"})
		app := newLocalStrategyTestApp(t, svc, user)
		app.Post("/legacy", func(c fiber.Ctx) error {
			legacy := fmt.Sprintf("t=tokendigest&id=%d&uid=%s&cat=%d", user.ID, user.UID, time.Now().Unix())
			return svc.SessionWrite(c, "auth", legacy)
		})

		sessionID := cookieValueByName(testRequest(t, app, httptest.NewRequest(http.MethodPost, "/legacy", nil)), "session_id")
		if resp := authenticateWithSession(t, app, sessionID); resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("expected a pre-upgrade session to stay anonymous, got %d", resp.StatusCode)
		}
	})

	t.Run("signs out the session of a deleted user", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		user := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "deleted@example.com", Password: "Password1!"})
		app := newLocalStrategyTestApp(t, svc, user)
		sessionID := signInForTest(t, app)
		ageSnapshotForTest(t, svc, sessionID)

		if err := svc.DB.Unscoped().Delete(&user).Error; err != nil {
			t.Fatalf("delete user: %v", err)
		}

		if resp := authenticateWithSession(t, app, sessionID); resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
		if raw := sessionValueForTest(t, svc, sessionID, authSessionKey); raw != "" {
			t.Fatalf("expected the signed-in user removed from the session, got %q", raw)
		}
	})
}

func TestRuntimeAuthenticateAndTakeUserByExtID(t *testing.T) {
	t.Run("Authenticate delegates to selected strategy", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		user := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "runtime@example.com", Password: "Password1!"})

		token, err := JWTCreateToken(svc.Env, user.Email, user.UID)
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
			if authed.ID != user.ID {
				t.Fatalf("expected %d, got %d", user.ID, authed.ID)
			}
			return c.SendStatus(fiber.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
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
		if err == nil || !strings.Contains(err.Error(), "authenticate user") {
			t.Fatalf("expected mapped auth error, got %v", err)
		}
	})
}

func TestDecodeAuthSnapshotQuery(t *testing.T) {
	t.Run("roundtrip encode decode", func(t *testing.T) {
		in := AuthSnapshot{
			Token:          "token",
			UserID:         42,
			UserUID:        "user_123",
			UserEmail:      "u@example.com",
			UserRole:       models.UserRoleAdmin,
			UserName:       "Ada",
			UserProfilePic: "/avatar.png",
			CachedAtUnix:   time.Now().Unix(),
		}

		out, ok := DecodeAuthSnapshotQuery(in.EncodeQuery())
		if !ok {
			t.Fatalf("expected decode success")
		}
		if out != in {
			t.Fatalf("decoded auth mismatch: got %+v", out)
		}
	})

	t.Run("fails for malformed cached timestamp", func(t *testing.T) {
		_, ok := DecodeAuthSnapshotQuery("id=1&uid=u1&cat=not-a-number")
		if ok {
			t.Fatalf("expected decode failure")
		}
	})

	t.Run("fails when required fields missing", func(t *testing.T) {
		_, ok := DecodeAuthSnapshotQuery("id=1&cat=123")
		if ok {
			t.Fatalf("expected decode failure when user uid is missing")
		}

		_, ok = DecodeAuthSnapshotQuery("uid=u1&cat=123")
		if ok {
			t.Fatalf("expected decode failure when user id is missing")
		}
	})
}

func TestAuthSnapshotFreshness(t *testing.T) {
	now := time.Unix(time.Now().Unix(), 0)

	t.Run("fresh at the hydration ttl", func(t *testing.T) {
		s := AuthSnapshot{UserID: 1, UserUID: "u1", CachedAtUnix: now.Add(-authSessionHydrationTTL).Unix()}
		if !s.isFresh(now) {
			t.Fatalf("expected snapshot fresh at ttl boundary")
		}
	})

	t.Run("stale past the hydration ttl", func(t *testing.T) {
		s := AuthSnapshot{UserID: 1, UserUID: "u1", CachedAtUnix: now.Add(-authSessionHydrationTTL - time.Second).Unix()}
		if s.isFresh(now) {
			t.Fatalf("expected snapshot stale past ttl")
		}
	})
}

// newLocalStrategyTestApp routes POST /signin to SignIn as user (?remember=1
// for remember me), GET / to Authenticate and POST /touch to a session write.
func newLocalStrategyTestApp(t *testing.T, svc *service, user models.User) *fiber.App {
	t.Helper()

	strategy := NewLocalStrategy(svc)
	app := fiber.New()
	app.Post("/signin", func(c fiber.Ctx) error {
		if err := strategy.SignIn(c, user, c.Query("remember") == "1"); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}
		return c.SendStatus(fiber.StatusNoContent)
	})
	app.Get("/", func(c fiber.Ctx) error {
		authed, err := strategy.Authenticate(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
		}
		c.Set("X-User-Name", authed.Name)
		return c.SendString(authed.UID)
	})
	app.Post("/touch", func(c fiber.Ctx) error {
		if err := svc.SessionWrite(c, "touched", "1"); err != nil {
			return err
		}
		return c.SendStatus(fiber.StatusNoContent)
	})

	return app
}

func signInForTest(t *testing.T, app *fiber.App) string {
	t.Helper()

	sessionID := cookieValueByName(testRequest(t, app, httptest.NewRequest(http.MethodPost, "/signin", nil)), "session_id")
	if sessionID == "" {
		t.Fatalf("expected session_id after sign in")
	}

	return sessionID
}

func ageSnapshotForTest(t *testing.T, svc *service, sessionID string) {
	t.Helper()

	editSnapshotForTest(t, svc, sessionID, func(s *AuthSnapshot) {
		s.CachedAtUnix -= int64((authSessionHydrationTTL + time.Minute) / time.Second)
	})
}

// sessionValueForTest returns the string stored under key in the session.
func sessionValueForTest(t *testing.T, svc *service, sessionID, key string) string {
	t.Helper()

	sess, err := svc.SessionStore.GetByID(t.Context(), sessionID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	defer sess.Release()

	raw, _ := sess.Get(key).(string)
	return raw
}

func snapshotForTest(t *testing.T, svc *service, sessionID string) AuthSnapshot {
	t.Helper()

	snapshot, ok := DecodeAuthSnapshotQuery(sessionValueForTest(t, svc, sessionID, authSessionKey))
	if !ok {
		t.Fatalf("expected a signed-in session")
	}

	return snapshot
}

// editSnapshotForTest changes the signed-in user stored in the session.
func editSnapshotForTest(t *testing.T, svc *service, sessionID string, edit func(*AuthSnapshot)) {
	t.Helper()

	snapshot := snapshotForTest(t, svc, sessionID)
	edit(&snapshot)

	sess, err := svc.SessionStore.GetByID(t.Context(), sessionID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	defer sess.Release()

	sess.Set(authSessionKey, snapshot.EncodeQuery())
	if err := sess.Save(); err != nil {
		t.Fatalf("save session: %v", err)
	}
}

func sessionTokenClaimsForTest(t *testing.T, svc *service, sessionID string) jwt.MapClaims {
	t.Helper()

	claims, err := decodeJWTToken(svc.Env, snapshotForTest(t, svc, sessionID).Token)
	if err != nil {
		t.Fatalf("decode session token: %v", err)
	}

	return claims
}

func signTokenForTest(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return token
}

func authenticateWithSession(t *testing.T, app *fiber.App, sessionID string) *http.Response {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
	return testRequest(t, app, req)
}

func testRequest(t *testing.T, app *fiber.App, req *http.Request) *http.Response {
	t.Helper()

	resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	return resp
}

func responseBody(t *testing.T, resp *http.Response) string {
	t.Helper()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	return string(body)
}

func countCookies(resp *http.Response, name string) int {
	n := 0
	for _, c := range resp.Cookies() {
		if c.Name == name {
			n++
		}
	}

	return n
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
