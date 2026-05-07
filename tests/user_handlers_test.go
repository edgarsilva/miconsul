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
	usersvc "miconsul/internal/services/user"

	"gorm.io/gorm"
)

func TestUserHandlers(t *testing.T) {
	h := newTestHarness(t)
	admin := h.createUser(models.UserRoleAdmin)
	regular := h.createUser(models.UserRoleUser)

	t.Run("profile update persists and redirects", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{
			method:    http.MethodPost,
			path:      "/profile",
			authToken: h.authToken(regular),
			body: url.Values{
				"name":  {"Updated Name"},
				"email": {"updated@example.com"},
				"phone": {"555-0199"},
			},
		})

		if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("expected redirect status, got %d", resp.StatusCode)
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
		if got := resp.Header.Get("HX-Redirect"); got != "/profile?toast=Profile picture removed&level=success" {
			t.Fatalf("expected HX-Redirect profile redirect, got %q", got)
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
