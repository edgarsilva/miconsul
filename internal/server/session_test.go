package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionConfigDefaults(t *testing.T) {
	t.Chdir(t.TempDir())

	cfg := sessionConfig("")
	if cfg.Database != defaultSessionDBPath {
		t.Fatalf("expected default session db path %q, got %q", defaultSessionDBPath, cfg.Database)
	}
	if cfg.Table != "fiber_storage" {
		t.Fatalf("expected default session table name, got %q", cfg.Table)
	}
	if _, err := os.Stat(filepath.Dir(defaultSessionDBPath)); err != nil {
		t.Fatalf("expected session db directory to be created: %v", err)
	}
}

func TestSessionConfigPaths(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "file path", path: filepath.Join(dir, "nested", "custom.sqlite"), want: filepath.Join(dir, "nested", "custom.sqlite")},
		{name: "directory path", path: dir, want: filepath.Join(dir, sessionDBFileName)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sessionConfig(tt.path).Database; got != tt.want {
				t.Fatalf("sessionConfig(%q).Database = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
