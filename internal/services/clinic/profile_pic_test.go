package clinic

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
)

func TestSaveProfilePicToDisk(t *testing.T) {
	t.Run("missing file returns ErrProfilePicNotProvided", func(t *testing.T) {
		runClinicCtx(t, "", nil, func(c fiber.Ctx) {
			_, err := SaveProfilePicToDisk(c, models.Clinic{UID: "cln_1"})
			if err != ErrProfilePicNotProvided {
				t.Fatalf("expected ErrProfilePicNotProvided, got %v", err)
			}
		})
	})

	t.Run("missing clinic id returns error", func(t *testing.T) {
		runClinicCtx(t, "photo.png", []byte("img"), func(c fiber.Ctx) {
			_, err := SaveProfilePicToDisk(c, models.Clinic{})
			if err == nil {
				t.Fatalf("expected missing clinic id to fail")
			}
		})
	})

	t.Run("rejects invalid sanitized filename", func(t *testing.T) {
		runClinicCtx(t, "..", []byte("img"), func(c fiber.Ctx) {
			_, err := SaveProfilePicToDisk(c, models.Clinic{UID: "cln_1"})
			if err == nil {
				t.Fatalf("expected invalid filename error")
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
