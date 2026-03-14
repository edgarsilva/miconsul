package jobs

import (
	"testing"

	"miconsul/internal/lib/appenv"
)

func TestNew(t *testing.T) {
	t.Run("returns error for nil env", func(t *testing.T) {
		runtime, err := New(nil)
		if err == nil {
			t.Fatal("expected error")
		}
		if runtime != nil {
			t.Fatal("expected nil runtime")
		}
	})

	t.Run("returns disabled runtime when jobs are disabled", func(t *testing.T) {
		runtime, err := New(&appenv.Env{JobsEnabled: false})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if runtime == nil {
			t.Fatal("expected runtime")
		}
		if runtime.Enabled() {
			t.Fatal("expected runtime disabled")
		}
		if err := runtime.Shutdown(); err != nil {
			t.Fatalf("unexpected shutdown error: %v", err)
		}
	})
}
