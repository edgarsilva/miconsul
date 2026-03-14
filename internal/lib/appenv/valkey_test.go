package appenv

import "testing"

func TestValkeyAddress(t *testing.T) {
	t.Run("nil env returns empty string", func(t *testing.T) {
		var env *Env
		if got := env.ValkeyAddress(); got != "" {
			t.Fatalf("ValkeyAddress() = %q, want empty", got)
		}
	})

	t.Run("uses defaults when host and port are missing", func(t *testing.T) {
		env := &Env{}
		if got := env.ValkeyAddress(); got != "127.0.0.1:6379" {
			t.Fatalf("ValkeyAddress() = %q, want %q", got, "127.0.0.1:6379")
		}
	})

	t.Run("trims host and uses configured port", func(t *testing.T) {
		env := &Env{ValkeyHost: "  valkey  ", ValkeyPort: 6381}
		if got := env.ValkeyAddress(); got != "valkey:6381" {
			t.Fatalf("ValkeyAddress() = %q, want %q", got, "valkey:6381")
		}
	})
}
