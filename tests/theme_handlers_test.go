package tests

import (
	"net/http"
	"strings"
	"testing"
)

func TestThemeHandlers(t *testing.T) {
	h := newTestHarness(t)

	t.Run("toggle returns theme button markup", func(t *testing.T) {
		resp, body := h.doRequest(requestOptions{method: http.MethodPost, path: "/theme/toggle", htmx: true})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for theme toggle, got %d", resp.StatusCode)
		}
		if !strings.Contains(body, "theme_toggle") {
			t.Fatalf("expected response body to include theme toggle markup")
		}
	})

	t.Run("toggle sets theme cookie", func(t *testing.T) {
		resp, _ := h.doRequest(requestOptions{method: http.MethodPost, path: "/theme/toggle", htmx: true})
		setCookie := resp.Header.Get("Set-Cookie")
		if setCookie == "" || !strings.Contains(setCookie, "theme=") {
			t.Fatalf("expected Set-Cookie header for theme, got %q", setCookie)
		}
	})
}
