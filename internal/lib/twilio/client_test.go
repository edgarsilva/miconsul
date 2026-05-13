package twilio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestPostFormSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/test"; got != want {
			t.Fatalf("unexpected path: %s", got)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "AC123" || pass != "token" {
			t.Fatalf("unexpected basic auth")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := New(Config{AccountSID: "AC123", AuthToken: "token", APIBaseURL: server.URL})
	body, err := client.PostForm(context.Background(), "/test", url.Values{"x": []string{"1"}})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected body: %s", string(body))
	}
}

func TestPostFormValidationErrors(t *testing.T) {
	t.Parallel()

	_, err := (*Client)(nil).PostForm(context.Background(), "/x", url.Values{})
	if err == nil {
		t.Fatalf("expected nil client error")
	}

	clientNoSID := New(Config{AuthToken: "token"})
	if _, err := clientNoSID.PostForm(context.Background(), "/x", url.Values{}); err == nil {
		t.Fatalf("expected missing sid error")
	}

	clientNoToken := New(Config{AccountSID: "AC123"})
	if _, err := clientNoToken.PostForm(context.Background(), "/x", url.Values{}); err == nil {
		t.Fatalf("expected missing token error")
	}
}

func TestPostFormProviderError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	client := New(Config{AccountSID: "AC123", AuthToken: "token", APIBaseURL: server.URL})
	_, err := client.PostForm(context.Background(), "/x", url.Values{})
	if err == nil {
		t.Fatalf("expected provider error")
	}
	if !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("expected status code in error, got %v", err)
	}
}
