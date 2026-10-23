package tests

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/models"
)

func withLogto(env *appenv.Env) {
	env.LogtoURL = "https://logto.example.com"
	env.LogtoAppID = "appid"
	env.LogtoAppSecret = "secret"
	env.LogtoResource = "https://api.example.com"
}

func hasAuthCookie(resp *http.Response) bool {
	for _, ck := range resp.Cookies() {
		if ck.Name == "Auth" && ck.Value != "" {
			return true
		}
	}
	return false
}

func TestLocalAuthDisabledWithLogto(t *testing.T) {
	h := newTestHarness(t, withLogto)
	admin := h.createAuthUser("", "Password1!", models.UserRoleAdmin)
	creds := url.Values{"email": {admin.Email}, "password": {"Password1!"}}

	t.Run("local auth endpoints are not served", func(t *testing.T) {
		for _, opt := range []requestOptions{
			{method: http.MethodPost, path: "/signin", body: creds},
			{method: http.MethodPost, path: "/api/auth/signin", body: creds},
			{method: http.MethodPost, path: "/signup", body: url.Values{"email": {"new@example.com"}, "password": {"Password1!"}}},
			{method: http.MethodGet, path: "/signup/confirm/sometoken"},
			{method: http.MethodPost, path: "/resetpassword", body: url.Values{"email": {admin.Email}}},
			{method: http.MethodGet, path: "/resetpassword/change/sometoken"},
			{method: http.MethodPost, path: "/resetpassword/change", body: url.Values{"token": {"sometoken"}, "password": {"Password1!"}}},
		} {
			resp, _ := h.doRequest(opt)
			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				t.Errorf("%s %s: status %d, want it refused", opt.method, opt.path, resp.StatusCode)
			}
			if hasAuthCookie(resp) {
				t.Errorf("%s %s: issued an Auth cookie", opt.method, opt.path)
			}
		}
	})

	t.Run("sign-up and reset go to Logto", func(t *testing.T) {
		for _, path := range []string{"/signup", "/resetpassword"} {
			resp, _ := h.doRequest(requestOptions{method: http.MethodGet, path: path})
			if got := resp.Header.Get("Location"); got != "/logto/signin" {
				t.Errorf("GET %s: redirect = %q (status %d), want /logto/signin", path, got, resp.StatusCode)
			}
		}
	})

	t.Run("signed-out page links to Logto without a password form", func(t *testing.T) {
		resp, body := h.doRequest(requestOptions{method: http.MethodGet, path: "/signin?logged_out=1"})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		if !strings.Contains(body, `href="/logto/signin"`) {
			t.Errorf("expected a link to /logto/signin")
		}
		if strings.Contains(body, `name="password"`) {
			t.Errorf("expected no password field")
		}
	})
}

func TestLocalAuthEnabledWithoutLogto(t *testing.T) {
	h := newTestHarness(t)

	resp, body := h.doRequest(requestOptions{method: http.MethodGet, path: "/signin?logged_out=1"})
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, `name="password"`) {
		t.Fatalf("expected the password form, status %d", resp.StatusCode)
	}

	for _, path := range []string{"/signup", "/resetpassword"} {
		resp, _ := h.doRequest(requestOptions{method: http.MethodGet, path: path})
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s: status = %d, want 200", path, resp.StatusCode)
		}
	}
}
