package appenv

import "testing"

func TestEnvAppURL(t *testing.T) {
	t.Run("returns base url when no path provided", func(t *testing.T) {
		env := &Env{AppProtocol: "https", AppDomain: "example.com"}
		if got := env.AppURL(); got != "https://example.com/" {
			t.Fatalf("AppURL() = %q, want %q", got, "https://example.com/")
		}
	})

	t.Run("joins multiple path segments", func(t *testing.T) {
		env := &Env{AppProtocol: "https", AppDomain: "example.com"}
		if got := env.AppURL("appointments", "new"); got != "https://example.com/appointments/new" {
			t.Fatalf("AppURL() = %q, want %q", got, "https://example.com/appointments/new")
		}
	})

	t.Run("trims protocol and domain input", func(t *testing.T) {
		env := &Env{AppProtocol: "  http ", AppDomain: " localhost:3000 "}
		if got := env.AppURL("dashboard"); got != "http://localhost:3000/dashboard" {
			t.Fatalf("AppURL() = %q, want %q", got, "http://localhost:3000/dashboard")
		}
	})

	t.Run("returns base url when path join fails", func(t *testing.T) {
		env := &Env{AppProtocol: "https", AppDomain: "example.com"}
		if got := env.AppURL("%zz"); got != "https://example.com" {
			t.Fatalf("AppURL() = %q, want %q", got, "https://example.com")
		}
	})

	t.Run("returns empty on nil env", func(t *testing.T) {
		var env *Env
		if got := env.AppURL("appointments"); got != "" {
			t.Fatalf("AppURL() = %q, want empty string", got)
		}
	})
}
