package valkey

import (
	"testing"

	"miconsul/internal/lib/appenv"
)

func TestNewConfig(t *testing.T) {
	t.Run("returns error for nil env", func(t *testing.T) {
		cfg, err := NewConfig(nil)
		if err == nil {
			t.Fatal("expected error")
		}
		if cfg != (Config{}) {
			t.Fatal("expected empty config on error")
		}
	})

	t.Run("returns valkey config from env", func(t *testing.T) {
		env := &appenv.Env{
			ValkeyHost:     "valkey",
			ValkeyPort:     6380,
			ValkeyPassword: "secret",
			ValkeyDB:       3,
		}

		cfg, err := NewConfig(env)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Address != "valkey:6380" {
			t.Fatalf("address = %q, want %q", cfg.Address, "valkey:6380")
		}
		if cfg.Password != "secret" {
			t.Fatalf("password = %q, want %q", cfg.Password, "secret")
		}
		if cfg.DB != 3 {
			t.Fatalf("db = %d, want %d", cfg.DB, 3)
		}
	})
}
