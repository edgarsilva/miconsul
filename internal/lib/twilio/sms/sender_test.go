package sms

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
		if values.Get("To") != "+5215512345678" {
			t.Fatalf("unexpected To: %s", values.Get("To"))
		}
		if values.Get("From") != "+14155238886" {
			t.Fatalf("unexpected From: %s", values.Get("From"))
		}
		if values.Get("Body") != "Your appointment is today at 3:00 PM." {
			t.Fatalf("unexpected Body: %s", values.Get("Body"))
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"sid":"SM123"}`))
	}))
	defer server.Close()

	sender := NewSender(Config{
		From:       "+14155238886",
		AccountSID: "AC123",
		AuthToken:  "token",
		APIBaseURL: server.URL,
	})

	err := sender.Send(context.Background(), Message{
		To:   "+52 155-123(45678)",
		Body: "Your appointment is today at 3:00 PM.",
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestSendValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		sender *Sender
		msg    Message
	}{
		{name: "nil sender", sender: nil, msg: Message{To: "+1", Body: "ok"}},
		{name: "missing from", sender: NewSender(Config{AccountSID: "AC123", AuthToken: "token"}), msg: Message{To: "+1", Body: "ok"}},
		{name: "missing to", sender: NewSender(Config{From: "+1", AccountSID: "AC123", AuthToken: "token"}), msg: Message{To: "", Body: "ok"}},
		{name: "invalid to", sender: NewSender(Config{From: "+1", AccountSID: "AC123", AuthToken: "token"}), msg: Message{To: "abc", Body: "ok"}},
		{name: "missing body", sender: NewSender(Config{From: "+1", AccountSID: "AC123", AuthToken: "token"}), msg: Message{To: "+1", Body: ""}},
		{name: "missing sid", sender: NewSender(Config{From: "+1", AuthToken: "token"}), msg: Message{To: "+1", Body: "ok"}},
		{name: "missing token", sender: NewSender(Config{From: "+1", AccountSID: "AC123"}), msg: Message{To: "+1", Body: "ok"}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.sender.Send(context.Background(), tt.msg); err == nil {
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

	sender := NewSender(Config{
		From:       "+14155238886",
		AccountSID: "AC123",
		AuthToken:  "token",
		APIBaseURL: server.URL,
	})

	err := sender.Send(context.Background(), Message{To: "+5215512345678", Body: "hello"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("expected status error, got %v", err)
	}
}

func TestNormalizePhone(t *testing.T) {
	t.Parallel()

	if got := normalizePhone("+52 155-123(45678)"); got != "+5215512345678" {
		t.Fatalf("unexpected normalized phone: %s", got)
	}
	if got := normalizePhone("3121014574"); got != "+523121014574" {
		t.Fatalf("expected MX default country code, got: %s", got)
	}
	if got := normalizePhone("5213121014574"); got != "+5213121014574" {
		t.Fatalf("expected explicit 52 country code preserved, got: %s", got)
	}
}
