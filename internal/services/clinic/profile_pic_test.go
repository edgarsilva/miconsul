package clinic

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"miconsul/internal/lib/profilepic"
	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
)

func TestSaveProfilePicToDisk(t *testing.T) {
	t.Run("missing file returns profilepic.ErrNotProvided", func(t *testing.T) {
		runClinicCtx(t, "", nil, func(c fiber.Ctx) {
			_, err := SaveProfilePicToDisk(c, models.Clinic{UID: "cln_1"}, t.TempDir())
			if !errors.Is(err, profilepic.ErrNotProvided) {
				t.Fatalf("expected profilepic.ErrNotProvided, got %v", err)
			}
		})
	})

	t.Run("missing clinic id returns error", func(t *testing.T) {
		runClinicCtx(t, "photo.png", []byte("img"), func(c fiber.Ctx) {
			_, err := SaveProfilePicToDisk(c, models.Clinic{}, t.TempDir())
			if err == nil {
				t.Fatalf("expected missing clinic id to fail")
			}
		})
	})

	t.Run("rejects a non-image filename", func(t *testing.T) {
		runClinicCtx(t, "..", []byte("img"), func(c fiber.Ctx) {
			_, err := SaveProfilePicToDisk(c, models.Clinic{UID: "cln_1"}, t.TempDir())
			if !errors.Is(err, profilepic.ErrInvalidFilename) {
				t.Fatalf("expected ErrInvalidFilename, got %v", err)
			}
		})
	})

	t.Run("stores the upload under a random name", func(t *testing.T) {
		dir := t.TempDir()
		runClinicCtx(t, "Logo.PNG", []byte("img"), func(c fiber.Ctx) {
			got, err := SaveProfilePicToDisk(c, models.Clinic{UID: "cln_1"}, dir)
			if err != nil {
				t.Fatalf("save: %v", err)
			}
			filename := strings.TrimPrefix(got, "/clinics/cln_1/profilepic/")
			if filename == got || !IsSafeProfilePicFilenameForClinic("cln_1", filename) || !strings.HasSuffix(filename, ".png") {
				t.Fatalf("unexpected url path %q", got)
			}
			if _, err := os.Stat(filepath.Join(dir, "clinics", filename)); err != nil {
				t.Fatalf("expected file on disk: %v", err)
			}
		})
	})
}

func runClinicCtx(t *testing.T, filename string, content []byte, fn func(c fiber.Ctx)) {
	t.Helper()

	app := fiber.New()
	app.Post("/upload", func(c fiber.Ctx) error {
		fn(c)
		return c.SendStatus(http.StatusNoContent)
	})

	body := bytes.Buffer{}
	writer := multipart.NewWriter(&body)
	if filename != "" {
		part, err := writer.CreateFormFile("profilePic", filename)
		if err != nil {
			t.Fatalf("create multipart file: %v", err)
		}
		if _, err = part.Write(content); err != nil {
			t.Fatalf("write multipart file: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("execute upload request: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected helper route status 204, got %d", resp.StatusCode)
	}
}
