package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/golang-jwt/jwt/v5"
	logto "github.com/logto-io/go/client"
)

func TestLogtoStrategyAuthenticate(t *testing.T) {
	tests := []struct {
		name         string
		expiresAt    time.Time
		refreshToken string
		wantErr      bool
		wantCalls    int32
	}{
		{name: "fresh access token signs in without calling logto", expiresAt: time.Now().Add(time.Hour), refreshToken: "refresh_1", wantCalls: 0},
		{name: "expired access token without refresh token is signed out", expiresAt: time.Now().Add(-time.Minute), wantErr: true, wantCalls: 0},
		{name: "expired access token with revoked refresh token is signed out", expiresAt: time.Now().Add(-time.Minute), refreshToken: "revoked", wantErr: true, wantCalls: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, user, logtoServer := newLogtoTestService(t)
			logtoServer.revoke = true
			cookies := seedLogtoSession(t, svc, user, tt.refreshToken, tt.expiresAt)

			authed, err := authenticateLogto(t, svc, cookies)
			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Fatalf("Authenticate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && authed.ID != user.ID {
				t.Fatalf("expected user %d, got %d", user.ID, authed.ID)
			}
			if got := logtoServer.calls.Load(); got != tt.wantCalls {
				t.Fatalf("expected %d calls to logto, got %d", tt.wantCalls, got)
			}
		})
	}
}

func TestLogtoStrategyAuthenticateRefresh(t *testing.T) {
	svc, user, logtoServer := newLogtoTestService(t)
	cookies := seedLogtoSession(t, svc, user, "refresh_1", time.Now().Add(-time.Minute))

	authed, err := authenticateLogto(t, svc, cookies)
	if err != nil {
		t.Fatalf("expected refreshed sign-in, got %v", err)
	}
	if authed.ID != user.ID {
		t.Fatalf("expected user %d, got %d", user.ID, authed.ID)
	}
	if got := logtoServer.calls.Load(); got != 3 {
		t.Fatalf("expected discovery, token and jwks calls, got %d", got)
	}

	if _, err := authenticateLogto(t, svc, cookies); err != nil {
		t.Fatalf("expected saved refreshed sign-in, got %v", err)
	}
	if got := logtoServer.calls.Load(); got != 3 {
		t.Fatalf("expected the refreshed token to be saved and reused, got %d calls", got)
	}
}

func TestHandleLogtoSignin(t *testing.T) {
	tests := []struct {
		name         string
		expiresAt    time.Time
		wantLocation string
	}{
		{name: "valid sign-in goes home", expiresAt: time.Now().Add(time.Hour), wantLocation: "/"},
		{name: "expired sign-in goes to logto", expiresAt: time.Now().Add(-time.Minute), wantLocation: "/oidc/auth"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, user, logtoServer := newLogtoTestService(t)
			logtoServer.revoke = true
			cookies := seedLogtoSession(t, svc, user, "revoked", tt.expiresAt)

			app := fiber.New()
			app.Get("/logto/signin", svc.HandleLogtoSignin)

			req := httptest.NewRequest(http.MethodGet, "/logto/signin", nil)
			for _, c := range cookies {
				req.AddCookie(c)
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}

			location := resp.Header.Get("Location")
			if tt.wantLocation == "/" && location != "/" {
				t.Fatalf("expected redirect home, got %q", location)
			}
			if tt.wantLocation != "/" && !strings.HasPrefix(location, logtoServer.srv.URL+tt.wantLocation) {
				t.Fatalf("expected redirect to logto sign-in, got %q", location)
			}
		})
	}
}

// fakeLogto is a Logto server whose token endpoint either rejects every
// refresh token (revoke) or issues new tokens.
type fakeLogto struct {
	srv    *httptest.Server
	key    *rsa.PrivateKey
	appID  string
	sub    string
	revoke bool
	calls  atomic.Int32
}

func newFakeLogto(t *testing.T, appID, sub string) *fakeLogto {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	fl := &fakeLogto{key: key, appID: appID, sub: sub}
	mux := http.NewServeMux()
	fl.srv = httptest.NewServer(mux)
	t.Cleanup(fl.srv.Close)

	mux.HandleFunc("/oidc/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		fl.calls.Add(1)
		writeTestJSON(w, http.StatusOK, map[string]string{
			"authorization_endpoint": fl.srv.URL + "/oidc/auth",
			"token_endpoint":         fl.srv.URL + "/oidc/token",
			"jwks_uri":               fl.srv.URL + "/oidc/jwks",
			"issuer":                 fl.issuer(),
		})
	})
	mux.HandleFunc("/oidc/token", func(w http.ResponseWriter, r *http.Request) {
		fl.calls.Add(1)
		if fl.revoke {
			writeTestJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
			return
		}

		writeTestJSON(w, http.StatusOK, map[string]any{
			"access_token":  "access_2",
			"refresh_token": "refresh_2",
			"id_token":      fl.idToken(t),
			"expires_in":    3600,
		})
	})
	mux.HandleFunc("/oidc/jwks", func(w http.ResponseWriter, r *http.Request) {
		fl.calls.Add(1)
		writeTestJSON(w, http.StatusOK, map[string][]map[string]string{"keys": {{
			"kty": "RSA",
			"kid": "test",
			"alg": "RS256",
			"use": "sig",
			"n":   base64.RawURLEncoding.EncodeToString(fl.key.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(fl.key.E)).Bytes()),
		}}})
	})

	return fl
}

func (fl *fakeLogto) issuer() string {
	return fl.srv.URL + "/oidc"
}

// idToken returns an ID token signed with the server's key.
func (fl *fakeLogto) idToken(t *testing.T) string {
	t.Helper()

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": fl.issuer(),
		"aud": fl.appID,
		"sub": fl.sub,
		"iat": now.Unix(),
		"exp": now.Add(time.Hour).Unix(),
	})
	token.Header["kid"] = "test"

	signed, err := token.SignedString(fl.key)
	if err != nil {
		t.Fatalf("sign id token: %v", err)
	}

	return signed
}

// newLogtoTestService returns an auth service with Logto pointed at a fake
// Logto server, and a user signed in to it.
func newLogtoTestService(t *testing.T) (*service, models.User, *fakeLogto) {
	t.Helper()

	svc := newAuthServiceForTests(t)
	user := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "logto@example.com", Password: "Password1!"})
	user.ExtID = "logto_sub_1"
	if err := svc.DB.Save(&user).Error; err != nil {
		t.Fatalf("set user ext id: %v", err)
	}

	logtoServer := newFakeLogto(t, "app_id", user.ExtID)

	svc.SessionStore = session.NewStore()
	svc.Env.AppProtocol = "http"
	svc.Env.AppDomain = "localhost:3000"
	svc.Env.LogtoURL = logtoServer.srv.URL
	svc.Env.LogtoAppID = logtoServer.appID
	svc.Env.LogtoAppSecret = "app_secret"
	svc.Env.LogtoResource = "https://api.example.com"

	return svc, user, logtoServer
}

// seedLogtoSession stores a Logto sign-in for user in a new session and
// returns the session cookies.
func seedLogtoSession(t *testing.T, svc *service, user models.User, refreshToken string, expiresAt time.Time) []*http.Cookie {
	t.Helper()

	idToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": user.ExtID})
	signedIDToken, err := idToken.SignedString([]byte(strings.Repeat("k", 32)))
	if err != nil {
		t.Fatalf("sign id token: %v", err)
	}
	accessToken := logto.AccessToken{Token: "access_1", ExpiresAt: expiresAt.Unix()}

	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		sess, err := svc.Session(c)
		if err != nil {
			return err
		}

		logtoClient, saveSess := NewLogtoClient(sess, LogtoConfig(svc.Env))
		logtoClient.SetIdToken(signedIDToken)
		logtoClient.SetRefreshToken(refreshToken)
		logtoClient.SaveAccessToken("@", accessToken) // key of the sign-in (no resource) access token
		return saveSess()
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("seed logto session: %v", err)
	}

	return resp.Cookies()
}

func authenticateLogto(t *testing.T, svc *service, cookies []*http.Cookie) (models.User, error) {
	t.Helper()

	var (
		authed  models.User
		authErr error
	)

	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		authed, authErr = NewLogtoStrategy(svc).Authenticate(c)
		return c.SendStatus(fiber.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	if _, err := app.Test(req); err != nil {
		t.Fatalf("request failed: %v", err)
	}

	return authed, authErr
}

func writeTestJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func TestHandleLogtoPage(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		wantText  string
	}{
		{name: "valid sign-in shows signed in", expiresAt: time.Now().Add(time.Hour), wantText: "You are logged in"},
		{name: "expired sign-in shows signed out", expiresAt: time.Now().Add(-time.Minute), wantText: "You are not logged in"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, user, logtoServer := newLogtoTestService(t)
			logtoServer.revoke = true
			cookies := seedLogtoSession(t, svc, user, "revoked", tt.expiresAt)

			app := fiber.New()
			app.Get("/logto", svc.HandleLogtoPage)

			req := httptest.NewRequest(http.MethodGet, "/logto", nil)
			for _, c := range cookies {
				req.AddCookie(c)
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if !strings.Contains(string(body), tt.wantText) {
				t.Fatalf("expected page to contain %q", tt.wantText)
			}
		})
	}
}

func TestLogtoStrategyAuthenticateCachesUser(t *testing.T) {
	t.Run("uses the cached user while fresh, then reloads it", func(t *testing.T) {
		svc, user, logtoServer := newLogtoTestService(t)
		cookies := seedLogtoSession(t, svc, user, "refresh_1", time.Now().Add(time.Hour))

		if _, err := authenticateLogto(t, svc, cookies); err != nil {
			t.Fatalf("expected sign-in, got %v", err)
		}
		if err := svc.DB.Delete(&user).Error; err != nil {
			t.Fatalf("delete user: %v", err)
		}

		authed, err := authenticateLogto(t, svc, cookies)
		if err != nil {
			t.Fatalf("expected the cached user, got %v", err)
		}
		if authed.ID != user.ID {
			t.Fatalf("expected cached user %d, got %d", user.ID, authed.ID)
		}

		editLogtoSession(t, svc, cookies, func(sess *session.Session) {
			raw, _ := sess.Get(authSessionKey).(string)
			snapshot, ok := DecodeAuthSnapshotQuery(raw)
			if !ok {
				t.Fatalf("expected a cached user in the session")
			}
			snapshot.CachedAtUnix = time.Now().Add(-authSessionHydrationTTL - time.Minute).Unix()
			sess.Set(authSessionKey, snapshot.EncodeQuery())
		})
		if _, err := authenticateLogto(t, svc, cookies); err == nil {
			t.Fatalf("expected a stale cache to reload the deleted user and fail")
		}
		if got := logtoServer.calls.Load(); got != 0 {
			t.Fatalf("expected no calls to logto, got %d", got)
		}
	})

	t.Run("ignores a cached user of another ID token", func(t *testing.T) {
		svc, user, _ := newLogtoTestService(t)
		other := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "other@example.com", Password: "Password1!"})
		cookies := seedLogtoSession(t, svc, user, "refresh_1", time.Now().Add(time.Hour))

		editLogtoSession(t, svc, cookies, func(sess *session.Session) {
			sess.Set(authSessionKey, newAuthSnapshot(other, "another.id.token").EncodeQuery())
		})

		authed, err := authenticateLogto(t, svc, cookies)
		if err != nil {
			t.Fatalf("expected sign-in, got %v", err)
		}
		if authed.ID != user.ID {
			t.Fatalf("expected the signed-in user %d, got %d", user.ID, authed.ID)
		}
	})

	t.Run("a cached user doesn't skip the access token check", func(t *testing.T) {
		svc, user, logtoServer := newLogtoTestService(t)
		logtoServer.revoke = true
		cookies := seedLogtoSession(t, svc, user, "revoked", time.Now().Add(-time.Minute))

		editLogtoSession(t, svc, cookies, func(sess *session.Session) {
			idToken, _ := sess.Get(logto.StorageKeyIdToken).(string)
			sess.Set(authSessionKey, newAuthSnapshot(user, idToken).EncodeQuery())
		})

		if _, err := authenticateLogto(t, svc, cookies); err == nil {
			t.Fatalf("expected a revoked sign-in to fail despite the cached user")
		}
	})
}

// editLogtoSession runs edit on the session of cookies and saves it.
func editLogtoSession(t *testing.T, svc *service, cookies []*http.Cookie, edit func(*session.Session)) {
	t.Helper()

	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		sess, err := svc.Session(c)
		if err != nil {
			return err
		}

		edit(sess)
		return sess.Save()
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	if _, err := app.Test(req); err != nil {
		t.Fatalf("edit logto session: %v", err)
	}
}
