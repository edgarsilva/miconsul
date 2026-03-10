package lib

import "testing"

func TestAppURL(t *testing.T) {
	t.Run("returns base url when no path provided", func(t *testing.T) {
		SetAppBaseURL("https", "example.com")
		if got := AppURL(); got != "https://example.com/" {
			t.Fatalf("AppURL() = %q, want %q", got, "https://example.com/")
		}
	})

	t.Run("joins multiple path segments", func(t *testing.T) {
		SetAppBaseURL("https", "example.com")
		if got := AppURL("appointments", "new"); got != "https://example.com/appointments/new" {
			t.Fatalf("AppURL() = %q, want %q", got, "https://example.com/appointments/new")
		}
	})

	t.Run("trims protocol and domain input", func(t *testing.T) {
		SetAppBaseURL("  http ", " localhost:3000 ")
		if got := AppURL("dashboard"); got != "http://localhost:3000/dashboard" {
			t.Fatalf("AppURL() = %q, want %q", got, "http://localhost:3000/dashboard")
		}
	})

	t.Run("returns base url when path join fails", func(t *testing.T) {
		SetAppBaseURL("https", "example.com")
		if got := AppURL("%zz"); got != "https://example.com" {
			t.Fatalf("AppURL() = %q, want %q", got, "https://example.com")
		}
	})
}
