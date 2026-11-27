package clinic

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"miconsul/internal/lib/objstore"
	"miconsul/internal/lib/objstore/objstoretest"
	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestClinicHandlerBranches(t *testing.T) {
	svc, user := newClinicServiceForTests(t)

	app := fiber.New()
	app.Get("/clinics/search", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleIndexSearch(c)
	})
	app.Get("/clinics/:id", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleEditPage(c)
	})
	app.Post("/clinics", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleCreate(c)
	})
	app.Post("/clinics/:id/patch", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleUpdate(c)
	})
	app.Post("/clinics/:id/delete", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleDelete(c)
	})

	resp1, err := app.Test(httptest.NewRequest(http.MethodGet, "/clinics/%20", nil))
	if err != nil || resp1.StatusCode != fiber.StatusNotFound {
		t.Fatalf("show with missing clinic expected 404, got status=%d err=%v", resp1.StatusCode, err)
	}

	resp2, err := app.Test(httptest.NewRequest(http.MethodGet, "/clinics/search?searchTerm=ab", nil))
	if err != nil || resp2.StatusCode != fiber.StatusOK {
		t.Fatalf("search short term expected 200 toast, got status=%d err=%v", resp2.StatusCode, err)
	}

	badJSON := httptest.NewRequest(http.MethodPost, "/clinics", strings.NewReader("{"))
	badJSON.Header.Set("Content-Type", "application/json")
	badJSON.Header.Set("HX-Request", "true")
	resp3, err := app.Test(badJSON)
	if err != nil || resp3.StatusCode != fiber.StatusNoContent {
		t.Fatalf("create invalid body expected 204 for htmx redirect, got status=%d err=%v", resp3.StatusCode, err)
	}

	resp4, err := app.Test(httptest.NewRequest(http.MethodPost, "/clinics/%20/patch", nil))
	if err != nil || resp4.StatusCode != fiber.StatusNotFound {
		t.Fatalf("update missing clinic expected 404, got status=%d err=%v", resp4.StatusCode, err)
	}

	resp5, err := app.Test(httptest.NewRequest(http.MethodPost, "/clinics/%20/delete", nil))
	if err != nil || resp5.StatusCode != fiber.StatusNotFound {
		t.Fatalf("delete missing clinic expected 404, got status=%d err=%v", resp5.StatusCode, err)
	}
}

func TestClinicHandlersHappyPaths(t *testing.T) {
	svc, user := newClinicServiceForTests(t)

	clinic := models.Clinic{UserID: user.ID, Name: "Alpha", Email: "alpha@example.com", Phone: "123"}
	if err := svc.CreateClinic(t.Context(), &clinic); err != nil {
		t.Fatalf("seed clinic: %v", err)
	}

	app := fiber.New()
	app.Get("/clinics/:id", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleEditPage(c)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/clinics/"+clinic.UID, nil))
	if err != nil || resp.StatusCode != fiber.StatusOK {
		t.Fatalf("show existing clinic expected 200, got status=%d err=%v", resp.StatusCode, err)
	}
}

func TestIndexSearchToasts(t *testing.T) {
	tests := []struct {
		name       string
		term       string
		htmx       bool
		wantStatus int
		wantToast  string
	}{
		{"short term", "ab", true, fiber.StatusOK, "at least 3 characters"},
		{"search without FTS table", "alpha", true, fiber.StatusInternalServerError, "couldn&#39;t be completed"},
		{"short term without htmx", "ab", false, fiber.StatusOK, "at least 3 characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, user := newClinicServiceForTests(t)

			app := fiber.New()
			app.Get("/clinics/search", func(c fiber.Ctx) error {
				c.Locals("current_user", user)
				return svc.HandleIndexSearch(c)
			})

			req := httptest.NewRequest(http.MethodGet, "/clinics/search?searchTerm="+tt.term, nil)
			if tt.htmx {
				req.Header.Set("HX-Request", "true")
			}
			resp, err := app.Test(req)
			if err != nil || resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got status=%d err=%v", tt.wantStatus, resp.StatusCode, err)
			}
			if got := resp.Header.Get("HX-Redirect"); got != "" {
				t.Errorf("search redirected to %q", got)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if !strings.Contains(string(body), `<hx-partial hx-target="#toast_region"`) || !strings.Contains(string(body), tt.wantToast) {
				t.Errorf("body = %s, want only a toast partial with %q", body, tt.wantToast)
			}
			if strings.Contains(string(body), `id="clinics_list"`) {
				t.Errorf("search replaced the list")
			}
		})
	}
}

func TestHandleUpdateClinicProfilePic(t *testing.T) {
	tests := []struct {
		name  string
		form  map[string]string
		saved bool
	}{
		{"valid update saves the uploaded picture", map[string]string{"name": "Renamed", "price": "0"}, true},
		{"invalid update keeps the old picture and writes no file", map[string]string{"name": "   ", "price": "0"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assetsDir := t.TempDir()
			picsDir := filepath.Join(assetsDir, "clinics")

			svc, user := newClinicServiceForTests(t)
			useDiskStore(t, svc, assetsDir)
			seed := models.Clinic{UserID: user.ID, Name: "Seed", ProfilePic: "old.png"}
			if err := svc.CreateClinic(t.Context(), &seed); err != nil {
				t.Fatalf("seed clinic: %v", err)
			}

			app := fiber.New()
			app.Patch("/clinics/:id", func(c fiber.Ctx) error {
				c.Locals("current_user", user)
				return svc.HandleUpdate(c)
			})

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			for name, value := range tt.form {
				if err := writer.WriteField(name, value); err != nil {
					t.Fatalf("write %s field: %v", name, err)
				}
			}
			part, err := writer.CreateFormFile("profilePic", "logo.png")
			if err != nil {
				t.Fatalf("create multipart file: %v", err)
			}
			if _, err := part.Write([]byte("png-data")); err != nil {
				t.Fatalf("write multipart file: %v", err)
			}
			if err := writer.Close(); err != nil {
				t.Fatalf("close multipart writer: %v", err)
			}

			req := httptest.NewRequest(http.MethodPatch, "/clinics/"+seed.UID, body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			req.Header.Set("HX-Request", "true")
			resp, err := app.Test(req)
			if err != nil || resp.StatusCode == fiber.StatusInternalServerError {
				t.Fatalf("update clinic failed: status=%d err=%v", resp.StatusCode, err)
			}

			raw, _ := io.ReadAll(resp.Body)
			if tt.saved && strings.Contains(string(raw), "hx-preserve") {
				t.Errorf("a saved form must not keep the selected picture file")
			}

			saved, err := svc.TakeClinicByID(t.Context(), user.ID, seed.UID)
			if err != nil {
				t.Fatalf("reload clinic: %v", err)
			}
			entries, err := os.ReadDir(picsDir)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("read pics dir: %v", err)
			}

			if tt.saved {
				if saved.ProfilePic == "" || saved.ProfilePic == "old.png" || len(entries) != 1 {
					t.Errorf("ProfilePic = %q with %d files, want the uploaded picture", saved.ProfilePic, len(entries))
				}
				return
			}
			if saved.ProfilePic != "old.png" || len(entries) != 0 {
				t.Errorf("ProfilePic = %q with %d files, want old.png and no files after an invalid update", saved.ProfilePic, len(entries))
			}
		})
	}
}

func TestAttachClinicPicRecordsAFailedSave(t *testing.T) {
	t.Chdir(t.TempDir())

	svc, user := newClinicServiceForTests(t)
	svc.Objects = objstoretest.Failing{Err: errors.New("store down")}
	recorder := tracetest.NewSpanRecorder()
	svc.Tracer = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)).Tracer("test")

	clinic := models.Clinic{UserID: user.ID, Name: "Seed", ProfilePic: "old.png"}
	if err := svc.CreateClinic(t.Context(), &clinic); err != nil {
		t.Fatalf("seed clinic: %v", err)
	}

	var got string
	app := fiber.New()
	app.Post("/pic", func(c fiber.Ctx) error {
		got = svc.attachClinicPic(c, user.ID, clinic)
		return c.SendStatus(fiber.StatusNoContent)
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("profilePic", "logo.png")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write([]byte("png-data")); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/pic", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if _, err := app.Test(req); err != nil {
		t.Fatalf("attach picture: %v", err)
	}

	if got != "old.png" {
		t.Errorf("picture = %q, want the current one kept after a failed save", got)
	}
	spans := recorder.Ended()
	if len(spans) != 1 || spans[0].Name() != "clinic/attachPic" {
		t.Fatalf("spans = %d, want one clinic/attachPic span", len(spans))
	}
	if spans[0].Status().Code != codes.Error || len(spans[0].Events()) == 0 {
		t.Errorf("span status = %v with %d events, want an error with the exception recorded", spans[0].Status().Code, len(spans[0].Events()))
	}
}

func TestAttachClinicPicRecordsAFailedDBUpdate(t *testing.T) {
	svc, user := newClinicServiceForTests(t)
	useDiskStore(t, svc, t.TempDir())
	recorder := tracetest.NewSpanRecorder()
	svc.Tracer = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)).Tracer("test")

	clinic := models.Clinic{UserID: user.ID, Name: "Seed", ProfilePic: "old.png"}
	if err := svc.CreateClinic(t.Context(), &clinic); err != nil {
		t.Fatalf("seed clinic: %v", err)
	}

	var got string
	app := fiber.New()
	app.Post("/pic", func(c fiber.Ctx) error {
		got = svc.attachClinicPic(c, user.ID+1, clinic)
		return c.SendStatus(fiber.StatusNoContent)
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("profilePic", "logo.png")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write([]byte("png-data")); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/pic", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if _, err := app.Test(req); err != nil {
		t.Fatalf("attach picture: %v", err)
	}

	if got != "old.png" {
		t.Errorf("picture = %q, want the current one kept after a failed update", got)
	}
	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("spans = %d, want one clinic/attachPic span", len(spans))
	}
	if status := spans[0].Status(); status.Code != codes.Error || status.Description != "attach profile pic failed" {
		t.Errorf("span status = %v %q, want the attach failure", status.Code, status.Description)
	}
}

func TestAttachClinicPicDeletesTheReplacedFile(t *testing.T) {
	svc, user := newClinicServiceForTests(t)
	assetsDir := t.TempDir()
	useDiskStore(t, svc, assetsDir)

	clinic := models.Clinic{UserID: user.ID, Name: "Alpha"}
	if err := svc.CreateClinic(t.Context(), &clinic); err != nil {
		t.Fatalf("seed clinic: %v", err)
	}

	oldName := clinic.UID + "_ppic_old.png"
	if err := svc.Objects.Put(t.Context(), ProfilePicKey(oldName), strings.NewReader("old"), 3, "image/png"); err != nil {
		t.Fatalf("write old picture: %v", err)
	}
	clinic.ProfilePic = "/clinics/" + clinic.UID + "/profilepic/" + oldName

	var got string
	app := fiber.New()
	app.Post("/pic", func(c fiber.Ctx) error {
		got = svc.attachClinicPic(c, user.ID, clinic)
		return c.SendStatus(fiber.StatusNoContent)
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("profilePic", "new.png")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write([]byte("new")); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/pic", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if _, err := app.Test(req); err != nil {
		t.Fatalf("attach picture: %v", err)
	}

	if got == "" || got == clinic.ProfilePic {
		t.Fatalf("picture = %q, want the new one", got)
	}
	if _, err := svc.Objects.Open(t.Context(), ProfilePicKey(oldName)); !errors.Is(err, objstore.ErrNotFound) {
		t.Errorf("old picture still exists: %v", err)
	}
	dir := filepath.Join(assetsDir, "clinics")
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Errorf("want only the new picture in %s, got %d entries err=%v", dir, len(entries), err)
	}
}
