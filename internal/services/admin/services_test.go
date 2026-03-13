package admin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileModelNameProviderListModelNames(t *testing.T) {
	tmpDir := t.TempDir()
	goodModel := filepath.Join(tmpDir, "patient.go")
	badModel := filepath.Join(tmpDir, "notes.txt")

	if err := os.WriteFile(goodModel, []byte("package model\n// --model:Patient\n"), 0o644); err != nil {
		t.Fatalf("write good model file: %v", err)
	}
	if err := os.WriteFile(badModel, []byte("no marker here\n"), 0o644); err != nil {
		t.Fatalf("write bad model file: %v", err)
	}

	provider := fileModelNameProvider{modelsDir: tmpDir}
	modelNames, err := provider.ListModelNames()
	if err != nil {
		t.Fatalf("list model names: %v", err)
	}
	if len(modelNames) != 1 || modelNames[0] != "Patient" {
		t.Fatalf("expected [Patient], got %#v", modelNames)
	}
}

func TestResolveModelsDir(t *testing.T) {
	modelsDir, err := resolveModelsDir()
	if err != nil {
		t.Fatalf("resolve models dir: %v", err)
	}

	if info, statErr := os.Stat(modelsDir); statErr != nil || !info.IsDir() {
		t.Fatalf("expected existing models dir, got %q err=%v", modelsDir, statErr)
	}
}
