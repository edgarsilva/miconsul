package tests

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"miconsul/internal/models"
)

func TestAuthHandlers(t *testing.T) {
	h := newTestHarness(t)
	u := h.createUser(model.UserRoleUser)
	token := h.authToken(u)
	authUser := h.createAuthUser("", "Password1!", model.UserRoleUser)

	t.Run("signin page renders when unauthenticated", func(t *testing.T) {
		resp, body := h.doRequest(requestOptions{method: http.MethodGet, path: "/signin"})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for signin page, got %d", resp.StatusCode)
		}
		if body == "" {
			t.Fatalf("expected non-empty signin page body")
		}
	})

	t.Run("signin page redirects when already authenticated", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{method: http.MethodGet, path: "/signin", authToken: token})
		if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("expected redirect status for authenticated signin page, got %d", resp.StatusCode)
		}
	})

	t.Run("api signin rejects blank credentials", func(t *testing.T) {
		resp, body := h.doRequest(requestOptions{
			method:     http.MethodPost,
			path:       "/api/auth/signin",
			accept:     "application/json",
			contentTyp: "application/x-www-form-urlencoded",
			body:       url.Values{"email": {""}, "password": {""}},
		})
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for blank API signin credentials, got %d", resp.StatusCode)
		}
		if !strings.Contains(body, "can't be blank") {
			t.Fatalf("expected blank credentials error message, got %q", body)
		}
	})

	t.Run("api signin rejects invalid credentials", func(t *testing.T) {
		resp, body := h.doRequest(requestOptions{
			method: http.MethodPost,
			path:   "/api/auth/signin",
			accept: "application/json",
			body: url.Values{
				"email":    {authUser.Email},
				"password": {"wrong-password"},
			},
		})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 for invalid API signin credentials, got %d", resp.StatusCode)
		}
		if !strings.Contains(body, "invalid credentials") {
			t.Fatalf("expected invalid credentials error message, got %q", body)
		}
	})

	t.Run("api signin success sets auth cookie", func(t *testing.T) {
		resp, body := h.doRequest(requestOptions{
			method: http.MethodPost,
			path:   "/api/auth/signin",
			accept: "application/json",
			body: url.Values{
				"email":    {authUser.Email},
				"password": {"Password1!"},
			},
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for successful API signin, got %d", resp.StatusCode)
		}
		if !strings.Contains(body, "\"ok\":true") {
			t.Fatalf("expected successful signin body, got %q", body)
		}
		if got := resp.Header.Get("Set-Cookie"); !strings.Contains(got, "Auth=") {
			t.Fatalf("expected Auth cookie to be set, got %q", got)
		}
	})

	t.Run("api protected rejects unauthenticated requests", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{method: http.MethodGet, path: "/api/auth/protected", accept: "application/json"})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 for unauthenticated protected endpoint, got %d", resp.StatusCode)
		}
	})

	t.Run("api protected returns user for authenticated requests", func(t *testing.T) {
		resp, body := h.doRequest(requestOptions{method: http.MethodGet, path: "/api/auth/protected", accept: "application/json", authToken: token})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for authenticated protected endpoint, got %d", resp.StatusCode)
		}
		if !strings.Contains(body, u.ID) {
			t.Fatalf("expected protected response to include current user id")
		}
	})

	t.Run("api validate enforces auth and accepts valid session", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{method: http.MethodPost, path: "/api/auth/validate", accept: "application/json"})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 for unauthenticated validate endpoint, got %d", resp.StatusCode)
		}

		resp, _ = h.doRequest(requestOptions{method: http.MethodPost, path: "/api/auth/validate", accept: "application/json", authToken: token})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for authenticated validate endpoint, got %d", resp.StatusCode)
		}
	})

	t.Run("logout htmx responds with redirect header", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{method: http.MethodPost, path: "/logout", htmx: true})
		if resp.StatusCode != http.StatusTemporaryRedirect {
			t.Fatalf("expected 307 for htmx logout, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("HX-Redirect"); got == "" {
			t.Fatalf("expected HX-Redirect header for htmx logout")
		}
	})

	t.Run("logout non-htmx redirects to signin", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{method: http.MethodPost, path: "/logout"})
		if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("expected redirect for non-htmx logout, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); !strings.Contains(got, "/signin") {
			t.Fatalf("expected redirect location to signin, got %q", got)
		}
	})
}
