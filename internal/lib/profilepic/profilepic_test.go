package profilepic

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestRandomName(t *testing.T) {
	re := regexp.MustCompile(`^u1_ppic_[0-9a-f]{32}\.png$`)

	a, err := RandomName("u1_ppic_", `C:\pics\Me.PNG`)
	if err != nil || !re.MatchString(a) {
		t.Fatalf("unexpected name %q err=%v", a, err)
	}

	b, _ := RandomName("u1_ppic_", "me.png")
	if a == b {
		t.Fatalf("names must differ, got %q twice", a)
	}

	for _, upload := range []string{"", "..", "photo", "x.exe", "a.png/../b.php"} {
		if _, err := RandomName("p_", upload); !errors.Is(err, ErrInvalidFilename) {
			t.Fatalf("expected ErrInvalidFilename for %q, got %v", upload, err)
		}
	}
}

func TestIsSafeFilename(t *testing.T) {
	cases := map[string]bool{
		"patient_ppic.png":       true,
		"a..b.png":               true,
		"":                       false,
		".":                      false,
		"..":                     false,
		"a/b.png":                false,
		`a\b.png`:                false,
		"a\x00.png":              false,
		"a?.png":                 false,
		"a b.png":                false,
		strings.Repeat("a", 256): false,
	}
	for name, want := range cases {
		if got := IsSafeFilename(name); got != want {
			t.Fatalf("IsSafeFilename(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestPath(t *testing.T) {
	dir := t.TempDir()

	got, err := Path(dir, "/patients", "a.png")
	if err != nil || got != filepath.Join(dir, "patients", "a.png") {
		t.Fatalf("unexpected path %q err=%v", got, err)
	}
	if info, err := os.Stat(filepath.Join(dir, "patients")); err != nil || !info.IsDir() {
		t.Fatalf("expected directory to exist: %v", err)
	}
	if _, err := Path(dir, "/patients", "../a.png"); !errors.Is(err, ErrInvalidFilename) {
		t.Fatalf("expected ErrInvalidFilename, got %v", err)
	}
	if _, err := Path(" ", "/patients", "a.png"); !errors.Is(err, ErrNoAssetsDir) {
		t.Fatalf("expected ErrNoAssetsDir, got %v", err)
	}
}

func TestFormFileError(t *testing.T) {
	if FormFileError(nil) != nil {
		t.Fatalf("nil must stay nil")
	}
	if !errors.Is(FormFileError(http.ErrMissingFile), ErrNotProvided) {
		t.Fatalf("missing file must map to ErrNotProvided")
	}
	if !errors.Is(FormFileError(errors.New("no uploaded file")), ErrNotProvided) {
		t.Fatalf("message match must map to ErrNotProvided")
	}
	if err := FormFileError(errors.New("boom")); errors.Is(err, ErrNotProvided) {
		t.Fatalf("other errors must not map to ErrNotProvided")
	}
}

func TestRemove(t *testing.T) {
	dir := t.TempDir()
	path, err := Path(dir, "/patients", "a.png")
	if err != nil {
		t.Fatalf("path: %v", err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := Remove(dir, "/patients", "a.png"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file still exists: %v", err)
	}
	if err := Remove(dir, "/patients", "a.png"); err != nil {
		t.Fatalf("removing a missing file must not fail, got %v", err)
	}
	if err := Remove(dir, "/patients", "../a.png"); !errors.Is(err, ErrInvalidFilename) {
		t.Fatalf("expected ErrInvalidFilename, got %v", err)
	}
	if err := Remove(" ", "/patients", "a.png"); !errors.Is(err, ErrNoAssetsDir) {
		t.Fatalf("expected ErrNoAssetsDir, got %v", err)
	}
}
