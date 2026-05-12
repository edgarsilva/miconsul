package twilio

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSendSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST")
		}
		if got, want := r.URL.Path, "/2010-04-01/Accounts/AC123/Messages.json"; got != want {
			t.Fatalf("unexpected path: %s", got)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "AC123" || pass != "token" {
			t.Fatalf("unexpected basic auth")
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		values, err := url.ParseQuery(string(bodyBytes))
		if err != nil {
			t.Fatalf("parse body: %v", err)
		}
		if values.Get("To") != "whatsapp:+5215512345678" {
			t.Fatalf("unexpected To: %s", values.Get("To"))
		}
		if values.Get("From") != "whatsapp:+14155238886" {
			t.Fatalf("unexpected From: %s", values.Get("From"))
		}
		if values.Get("Body") != "hello" {
			t.Fatalf("unexpected Body: %s", values.Get("Body"))
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"sid":"SM123"}`))
	}))
	defer server.Close()

	sender := New(Config{
		AccountSID:   "AC123",
		AuthToken:    "token",
		WhatsAppFrom: "+14155238886",
		APIBaseURL:   server.URL,
	})

	err := sender.Send(context.Background(), "+5215512345678", "hello")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestSendValidationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		sender *Sender
		to     string
		text   string
	}{
		{name: "nil sender", sender: nil, to: "+521", text: "x"},
		{name: "missing sid", sender: New(Config{AuthToken: "token", WhatsAppFrom: "+1"}), to: "+521", text: "x"},
		{name: "missing token", sender: New(Config{AccountSID: "AC123", WhatsAppFrom: "+1"}), to: "+521", text: "x"},
		{name: "missing from", sender: New(Config{AccountSID: "AC123", AuthToken: "token"}), to: "+521", text: "x"},
		{name: "missing to", sender: New(Config{AccountSID: "AC123", AuthToken: "token", WhatsAppFrom: "+1"}), to: "", text: "x"},
		{name: "missing text", sender: New(Config{AccountSID: "AC123", AuthToken: "token", WhatsAppFrom: "+1"}), to: "+521", text: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.sender.Send(context.Background(), tt.to, tt.text)
			if err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestSendProviderError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	sender := New(Config{
		AccountSID:   "AC123",
		AuthToken:    "token",
		WhatsAppFrom: "+14155238886",
		APIBaseURL:   server.URL,
	})

	err := sender.Send(context.Background(), "+5215512345678", "hello")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("expected status in error, got %v", err)
	}
}

func TestWithWhatsAppPrefix(t *testing.T) {
	t.Parallel()

	if got := withWhatsAppPrefix("+123"); got != "whatsapp:+123" {
		t.Fatalf("unexpected prefixed value: %s", got)
	}
	if got := withWhatsAppPrefix("whatsapp:+123"); got != "whatsapp:+123" {
		t.Fatalf("unexpected already prefixed value: %s", got)
	}
}
