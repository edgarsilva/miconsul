package mailer

import (
	"os"
	"strings"
	"testing"
)

func TestDialerCredentialsAndLocalization(t *testing.T) {
	t.Setenv("EMAIL_SENDER", "sender@example.com")
	t.Setenv("EMAIL_SECRET", "\"secret123\"")

	if got := dialerUsername(); got != "sender@example.com" {
		t.Fatalf("expected sender@example.com, got %q", got)
	}
	if got := dialerPassword(); got != "secret123" {
		t.Fatalf("expected unquoted secret, got %q", got)
	}

	if got := l("es-MX", "email.confirm_appointment_title"); got == "" {
		t.Fatalf("expected localized string for known key")
	}
	if got := l("xx-XX", "missing.key"); got != "missing.key" {
		t.Fatalf("expected fallback to key for unknown localization, got %q", got)
	}

	_ = os.Unsetenv("EMAIL_SENDER")
}

func TestStringAndURLHelpers(t *testing.T) {
	if got := keepChars("+52 (81) 1234-5678", "1234567890"); got != "528112345678" {
		t.Fatalf("unexpected keepChars result %q", got)
	}

	if got := removeChars("a-b_c.d", "-_."); got != "abcd" {
		t.Fatalf("unexpected removeChars result %q", got)
	}

	url := waURL("+52 (81) 1234-5678", "hello world")
	if !strings.Contains(url, "https://wa.me/528112345678?text=hello world") {
		t.Fatalf("unexpected whatsapp url %q", url)
	}
}
