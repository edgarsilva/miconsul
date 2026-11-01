package tests

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"miconsul/internal/models"
	"miconsul/internal/server/servertest"
	usersvc "miconsul/internal/services/user"

	"gorm.io/gorm"
)

func TestUserHandlers(t *testing.T) {
	h := newTestHarness(t)
	admin := h.createUser(models.UserRoleAdmin)
	regular := h.createUser(models.UserRoleUser)

	t.Run("user form back link fits who is editing", func(t *testing.T) {
		tests := []struct {
			name, path, token, wantHref string
		}{
			{"own profile", "/profile", h.authToken(regular), `href="/"`},
			{"admin editing another user", "/admin/users/" + regular.UID, h.authToken(admin), `href="/admin/users"`},
		}
		for _, tt := range tests {
			resp, body := h.doRequest(requestOptions{path: tt.path, authToken: tt.token})
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("%s: expected 200, got %d", tt.name, resp.StatusCode)
			}
			if !strings.Contains(body, `class="btn btn-secondary btn-sm" `+tt.wantHref) {
				t.Errorf("%s: expected the back button to link %s", tt.name, tt.wantHref)
			}
		}
	})

	t.Run("profile update persists and renders the profile section", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodPatch,
			path:      "/profile",
			authToken: h.authToken(regular),
			htmx:      true,
			body: url.Values{
				"name":  {"Updated Name"},
				"email": {"updated@example.com"},
				"phone": {"555-0199"},
			},
		})

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for htmx profile update, got %d", resp.StatusCode)
		}

		updated, err := gorm.G[models.User](h.db.GormDB()).Where("id = ?", regular.ID).Take(t.Context())
		if err != nil {
			t.Fatalf("load updated user: %v", err)
		}
		if updated.Name != "Updated Name" || updated.Email != "updated@example.com" || updated.Phone != "555-0199" {
			t.Fatalf("unexpected persisted profile: %+v", updated)
		}
	})

	t.Run("profile removepic htmx redirects back to profile", func(t *testing.T) {
		if _, err := gorm.G[models.User](h.db.GormDB()).Where("id = ?", regular.ID).Updates(t.Context(), models.User{ProfilePic: "/public/images/profile.png"}); err != nil {
			t.Fatalf("seed profile pic: %v", err)
		}

		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodDelete,
			path:      "/profile/avatar",
			authToken: h.authToken(regular),
			htmx:      true,
		})

		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204 for htmx removepic, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("HX-Redirect"); got != "/profile" {
			t.Fatalf("expected HX-Redirect profile redirect, got %q", got)
		}
		if msg, level := servertest.FlashToast(t, resp); msg != "Profile picture removed" || level != "success" {
			t.Fatalf("flash toast = %q %q, want the removed message", msg, level)
		}

		updated, err := gorm.G[models.User](h.db.GormDB()).Where("id = ?", regular.ID).Take(t.Context())
		if err != nil {
			t.Fatalf("load updated user: %v", err)
		}
		if updated.ProfilePic != "" {
			t.Fatalf("expected profile pic removed, got %q", updated.ProfilePic)
		}
	})

	t.Run("api users auth gating", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{method: http.MethodGet, path: "/api/users", accept: "application/json"})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 for unauthenticated request, got %d", resp.StatusCode)
		}

		resp, _ = h.doRequest(requestOptions{method: http.MethodGet, path: "/api/users", accept: "application/json", authToken: h.authToken(regular)})
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 for non-admin request, got %d", resp.StatusCode)
		}

		resp, body := h.doRequest(requestOptions{method: http.MethodGet, path: "/api/users", accept: "application/json", authToken: h.authToken(admin)})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for admin request, got %d", resp.StatusCode)
		}
		if body == "" {
			t.Fatalf("expected non-empty JSON body for admin /api/users")
		}
	})

	t.Run("profile pic preview stores tmp and can be served", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		if err := writer.WriteField("id", regular.UID); err != nil {
			t.Fatalf("write id field: %v", err)
		}
		part, err := writer.CreateFormFile("profilePic", "preview.jpg")
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := part.Write([]byte("preview-bits")); err != nil {
			t.Fatalf("write form file: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("close writer: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/profile/avatar/preview", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Accept", "text/html")
		req.Header.Set("HX-Request", "true")
		req.Header.Set("Authorization", "Bearer "+h.authToken(regular))

		resp, err := h.server.Test(req)
		if err != nil {
			t.Fatalf("preview request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 from profile preview, got %d", resp.StatusCode)
		}

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read preview body: %v", err)
		}
		expectedPath := "/profile/avatar/preview"
		if !strings.Contains(string(respBody), expectedPath) {
			t.Fatalf("expected preview img src in response body, got %q", string(respBody))
		}

		previewFilePath, err := usersvc.ProfilePicPath(regular.UID+"_preview", h.env.AssetsDir)
		if err != nil {
			t.Fatalf("resolve preview file path: %v", err)
		}
		if _, err := os.Stat(previewFilePath); err != nil {
			t.Fatalf("expected preview file in assets dir: %v", err)
		}

	})
}

func TestProfileUpdateRefreshesSignedInUser(t *testing.T) {
	h := newTestHarness(t)
	user := h.createAuthUser("", "Password1!", models.UserRoleUser)
	profilePic := "/public/images/before.png"
	if _, err := gorm.G[models.User](h.db.GormDB()).Where("id = ?", user.ID).Updates(t.Context(), models.User{ProfilePic: profilePic}); err != nil {
		t.Fatalf("seed profile pic: %v", err)
	}

	resp, _ := h.doRequest(requestOptions{
		method: http.MethodPost,
		path:   "/api/auth/signin",
		accept: "application/json",
		body:   url.Values{"email": {user.Email}, "password": {"Password1!"}},
	})
	sessionCookie := onlySessionCookie(t, resp)
	withSession := []*http.Cookie{{Name: sessionCookie.Name, Value: sessionCookie.Value}}

	resp, _ = h.doRequest(requestOptions{
		method:  http.MethodPatch,
		path:    "/profile",
		cookies: withSession,
		htmx:    true,
		body:    url.Values{"name": {"Renamed User"}, "email": {user.Email}},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for htmx profile update, got %d", resp.StatusCode)
	}

	resp, body := h.doRequest(requestOptions{method: http.MethodGet, path: "/api/auth/protected", accept: "application/json", cookies: withSession})
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "Renamed User") {
		t.Fatalf("expected the next request to show the saved name, got %d %q", resp.StatusCode, body)
	}

	resp, _ = h.doRequest(requestOptions{method: http.MethodDelete, path: "/profile/avatar", cookies: withSession, htmx: true})
	if resp.StatusCode >= http.StatusBadRequest {
		t.Fatalf("expected the profile picture removal to succeed, got %d", resp.StatusCode)
	}

	resp, body = h.doRequest(requestOptions{method: http.MethodGet, path: "/api/auth/protected", accept: "application/json", cookies: withSession})
	if resp.StatusCode != http.StatusOK || strings.Contains(body, profilePic) {
		t.Fatalf("expected the next request to show no profile picture, got %d %q", resp.StatusCode, body)
	}
}
