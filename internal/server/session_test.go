package server

import "testing"

func TestSessionConfigDefaults(t *testing.T) {
	cfg := sessionConfig("")
	if cfg.Database == "" {
		t.Fatalf("expected default session db path")
	}
	if cfg.Table != "fiber_storage" {
		t.Fatalf("expected default session table name, got %q", cfg.Table)
	}
}
