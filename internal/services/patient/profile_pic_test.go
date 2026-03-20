package patient

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
)

func TestCleanFilenameSegment(t *testing.T) {
	t.Run("keeps allowed chars and replaces spaces", func(t *testing.T) {
		got := cleanFilenameSegment("  my photo-1.png  ")
		if got != "my_photo-1.png" {
			t.Fatalf("expected sanitized filename, got %q", got)
		}
	})

	t.Run("drops disallowed chars", func(t *testing.T) {
		got := cleanFilenameSegment("../../we!rd?#name.jpg")
		if got != "....werdname.jpg" {
			t.Fatalf("expected invalid characters removed, got %q", got)
		}
	})

	t.Run("empty when only invalid chars", func(t *testing.T) {
		got := cleanFilenameSegment(" $$$ ")
		if got != "" {
			t.Fatalf("expected empty result, got %q", got)
		}
	})
}

func TestIsSafeFilename(t *testing.T) {
	cases := []struct {
		name     string
		filename string
		want     bool
	}{
		{name: "valid basic filename", filename: "patient_ppic.png", want: true},
		{name: "reject slash", filename: "a/b.png", want: false},
		{name: "reject backslash", filename: `a\\b.png`, want: false},
		{name: "reject dot", filename: ".", want: false},
		{name: "reject dotdot", filename: "..", want: false},
		{name: "reject null byte", filename: "a\x00.png", want: false},
		{name: "reject unsanitized chars", filename: "a?.png", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isSafeFilename(tc.filename)
			if got != tc.want {
				t.Fatalf("expected %v for %q, got %v", tc.want, tc.filename, got)
			}
		})
	}
}

func TestIsSafeProfilePicFilenameForPatient(t *testing.T) {
	if !IsSafeProfilePicFilenameForPatient("pat_123", "pat_123_ppic_photo.png") {
		t.Fatalf("expected matching patient-prefixed filename to be valid")
	}
	if IsSafeProfilePicFilenameForPatient("pat_123", "pat_999_ppic_photo.png") {
		t.Fatalf("expected mismatched patient prefix to be invalid")
	}
	if IsSafeProfilePicFilenameForPatient("", "pat_123_ppic_photo.png") {
		t.Fatalf("expected empty patient id to be invalid")
	}
}

func TestIsMissingProfilePicErr(t *testing.T) {
	if isMissingProfilePicErr(nil) {
		t.Fatalf("expected nil error to be non-missing")
	}
	if !isMissingProfilePicErr(http.ErrMissingFile) {
		t.Fatalf("expected http.ErrMissingFile to be missing")
	}
	if !isMissingProfilePicErr(errors.New("request is not multipart/form-data")) {
		t.Fatalf("expected multipart error message to be missing")
	}
	if isMissingProfilePicErr(errors.New("another error")) {
		t.Fatalf("expected unrelated error to be non-missing")
	}
}

func TestProfilePicFormFileErr(t *testing.T) {
	if !errors.Is(profilePicFormFileErr(http.ErrMissingFile), ErrProfilePicNotProvided) {
		t.Fatalf("expected missing file error to map to ErrProfilePicNotProvided")
	}

	err := profilePicFormFileErr(errors.New("broken multipart reader"))
	if errors.Is(err, ErrProfilePicNotProvided) {
		t.Fatalf("expected non-missing errors not to map to ErrProfilePicNotProvided")
	}
}

func TestProfilePicPath(t *testing.T) {
	t.Run("returns full path and creates parent dir", func(t *testing.T) {
		assetsDir := t.TempDir()
		path, err := ProfilePicPath("pat_1_ppic_photo.png", assetsDir)
		if err != nil {
			t.Fatalf("expected valid path, got error: %v", err)
		}

		expected := filepath.Join(assetsDir, patientsDir, "pat_1_ppic_photo.png")
		if path != expected {
			t.Fatalf("expected path %q, got %q", expected, path)
		}
	})

	t.Run("fails with unsafe filename", func(t *testing.T) {
		if _, err := ProfilePicPath("../../x.png", t.TempDir()); !errors.Is(err, ErrInvalidFilename) {
			t.Fatalf("expected ErrInvalidFilename, got %v", err)
		}
	})

	t.Run("fails with blank assets directory", func(t *testing.T) {
		if _, err := ProfilePicPath("pat_1_ppic_photo.png", "   "); err == nil {
			t.Fatalf("expected blank assets dir error")
		}
	})
}

func TestSaveProfilePicToDisk(t *testing.T) {
	t.Run("returns ErrProfilePicNotProvided when form file missing", func(t *testing.T) {
		app := fiber.New()
		app.Post("/", func(c fiber.Ctx) error {
			_, err := SaveProfilePicToDisk(c, model.Patient{ID: "pat_1"}, t.TempDir())
			if !errors.Is(err, ErrProfilePicNotProvided) {
				t.Fatalf("expected ErrProfilePicNotProvided, got %v", err)
			}
			return c.SendStatus(fiber.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(nil))
		if _, err := app.Test(req); err != nil {
			t.Fatalf("request failed: %v", err)
		}
	})

	t.Run("fails when patient id is blank", func(t *testing.T) {
		app := fiber.New()
		app.Post("/", func(c fiber.Ctx) error {
			_, err := SaveProfilePicToDisk(c, model.Patient{}, t.TempDir())
			if err == nil {
				t.Fatalf("expected missing patient id error")
			}
			return c.SendStatus(fiber.StatusNoContent)
		})

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("profilePic", "avatar.png")
		if err != nil {
			t.Fatalf("create multipart file: %v", err)
		}
		if _, err := part.Write([]byte("png-data")); err != nil {
			t.Fatalf("write multipart file: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("close multipart writer: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		if _, err := app.Test(req); err != nil {
			t.Fatalf("request failed: %v", err)
		}
	})
}
