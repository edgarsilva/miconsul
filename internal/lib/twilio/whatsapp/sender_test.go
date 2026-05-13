package whatsapp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSendTemplateSuccess(t *testing.T) {
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
		if values.Get("To") != "whatsapp:+5215512345678" {
			t.Fatalf("unexpected To: %s", values.Get("To"))
		}
		if values.Get("From") != "whatsapp:+14155238886" {
			t.Fatalf("unexpected From: %s", values.Get("From"))
		}
		if values.Get("ContentSid") != "HX123" {
			t.Fatalf("unexpected ContentSid: %s", values.Get("ContentSid"))
		}
		if values.Get("ContentVariables") != `{"1":"12/1","2":"3pm"}` {
			t.Fatalf("unexpected ContentVariables: %s", values.Get("ContentVariables"))
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"sid":"SM123"}`))
	}))
	defer server.Close()

	sender := NewSender(Config{
		WhatsAppFrom: "+14155238886",
		ContentSID:   "HX123",
		AccountSID:   "AC123",
		AuthToken:    "token",
		APIBaseURL:   server.URL,
	})

	err := sender.SendTemplate(context.Background(), TemplateMessage{
		To: "+5215512345678",
		Variables: map[string]string{
			"1": "12/1",
			"2": "3pm",
		},
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestSendTemplateValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		sender *Sender
		msg    TemplateMessage
	}{
		{name: "nil sender", sender: nil, msg: TemplateMessage{To: "+1"}},
		{name: "missing from", sender: NewSender(Config{ContentSID: "HX", AccountSID: "AC123", AuthToken: "token"}), msg: TemplateMessage{To: "+1"}},
		{name: "missing content sid", sender: NewSender(Config{WhatsAppFrom: "+1", AccountSID: "AC123", AuthToken: "token"}), msg: TemplateMessage{To: "+1"}},
		{name: "missing to", sender: NewSender(Config{WhatsAppFrom: "+1", ContentSID: "HX", AccountSID: "AC123", AuthToken: "token"}), msg: TemplateMessage{To: ""}},
		{name: "missing sid", sender: NewSender(Config{WhatsAppFrom: "+1", ContentSID: "HX", AuthToken: "token"}), msg: TemplateMessage{To: "+1"}},
		{name: "missing token", sender: NewSender(Config{WhatsAppFrom: "+1", ContentSID: "HX", AccountSID: "AC123"}), msg: TemplateMessage{To: "+1"}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.sender.SendTemplate(context.Background(), tt.msg); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestSendTemplateProviderError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	sender := NewSender(Config{
		WhatsAppFrom: "+14155238886",
		ContentSID:   "HX123",
		AccountSID:   "AC123",
		AuthToken:    "token",
		APIBaseURL:   server.URL,
	})

	err := sender.SendTemplate(context.Background(), TemplateMessage{To: "+5215512345678"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("expected status error, got %v", err)
	}
}

func TestWithWhatsAppPrefix(t *testing.T) {
	t.Parallel()

	if got := withWhatsAppPrefix("+123"); got != "whatsapp:+123" {
		t.Fatalf("unexpected prefixed value: %s", got)
	}
	if got := withWhatsAppPrefix("whatsapp:+123"); got != "whatsapp:+123" {
		t.Fatalf("unexpected already-prefixed value: %s", got)
	}
}
