package patient

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"miconsul/internal/lib/profilepic"
	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
)

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

func TestProfilePicKey(t *testing.T) {
	if got := ProfilePicKey("pat_1_ppic_photo.png"); got != "patients/pat_1_ppic_photo.png" {
		t.Fatalf("ProfilePicKey = %q", got)
	}
}

func TestSaveProfilePic(t *testing.T) {
	t.Run("returns profilepic.ErrNotProvided when form file missing", func(t *testing.T) {
		app := fiber.New()
		app.Post("/", func(c fiber.Ctx) error {
			_, err := SaveProfilePic(c, newTestStore(t, t.TempDir()), models.Patient{UID: "pat_1"})
			if !errors.Is(err, profilepic.ErrNotProvided) {
				t.Fatalf("expected profilepic.ErrNotProvided, got %v", err)
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
			_, err := SaveProfilePic(c, newTestStore(t, t.TempDir()), models.Patient{})
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
