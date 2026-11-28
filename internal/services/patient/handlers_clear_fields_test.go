package patient

import (
	"bytes"
	"errors"
	"html"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"slices"
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

func TestHandleUpdateSavesClearedFields(t *testing.T) {
	svc, user := newPatientServiceForTests(t)

	var full patientUpsertInput
	fillFields(&full)
	filled := full.toPatient(0, "", user.ID)

	notInForm := []string{"Line2", "Whatsapp", "Telegram", "Messenger", "Instagram", "Facebook"}
	seed := filled
	seed.ProfilePic = "old.png"
	for _, name := range notInForm {
		reflect.ValueOf(&seed).Elem().FieldByName(name).SetString("kept")
	}
	if err := svc.CreatePatient(t.Context(), &seed); err != nil {
		t.Fatalf("seed patient: %v", err)
	}
	before, err := svc.TakePatientByID(t.Context(), user.ID, seed.UID)
	if err != nil {
		t.Fatalf("load seeded patient: %v", err)
	}

	app := fiber.New()
	app.Patch("/patients/:id", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleUpdate(c)
	})

	requiredFieldsOnly := url.Values{"name": {"Renamed"}, "age": {"1"}, "phone": {"1"}}
	req := httptest.NewRequest(http.MethodPatch, "/patients/"+seed.UID, strings.NewReader(requiredFieldsOnly.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("update patient expected 204, got status=%d err=%v", resp.StatusCode, err)
	}

	saved, err := svc.TakePatientByID(t.Context(), user.ID, seed.UID)
	if err != nil {
		t.Fatalf("reload patient: %v", err)
	}
	assertFieldsCleared(t, filled, saved, "UserID", "Name", "Age", "Phone")
	for _, name := range notInForm {
		if got := reflect.ValueOf(saved).FieldByName(name).String(); got != "kept" {
			t.Errorf("%s = %q, want it kept because the form does not render it", name, got)
		}
	}
	if saved.ProfilePic != "old.png" {
		t.Errorf("ProfilePic = %q, want old.png kept when no picture is uploaded", saved.ProfilePic)
	}
	if !saved.UpdatedAt.After(before.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want after %v", saved.UpdatedAt, before.UpdatedAt)
	}
}

// fillFields sets every string, int and bool field of the struct ptr points to.
func fillFields(ptr any) {
	for _, field := range reflect.ValueOf(ptr).Elem().Fields() {
		switch field.Kind() {
		case reflect.String:
			field.SetString("1")
		case reflect.Int:
			field.SetInt(1)
		case reflect.Bool:
			field.SetBool(true)
		}
	}
}

// assertFieldsCleared fails for each field set in filled that got still holds.
func assertFieldsCleared(t *testing.T, filled, got any, skip ...string) {
	t.Helper()

	filledValue := reflect.ValueOf(filled)
	gotValue := reflect.ValueOf(got)
	for _, field := range reflect.VisibleFields(filledValue.Type()) {
		if field.Anonymous || !field.IsExported() || slices.Contains(skip, field.Name) {
			continue
		}
		if filledValue.FieldByIndex(field.Index).IsZero() {
			continue
		}
		if value := gotValue.FieldByIndex(field.Index); !value.IsZero() {
			t.Errorf("%s = %v, want empty after clearing the form", field.Name, value.Interface())
		}
	}
}

func TestHandleUpdateProfilePic(t *testing.T) {
	tests := []struct {
		name   string
		fields map[string]string
		saved  bool
	}{
		{"valid update saves the uploaded picture", map[string]string{"name": "Alpha", "age": "30", "phone": "111"}, true},
		{"invalid update keeps the old picture", map[string]string{"name": "Alpha"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, user := newPatientServiceForTests(t)
			assetsDir := t.TempDir()
			useDiskStore(t, svc, assetsDir)

			seed := models.Patient{UserID: user.ID, Name: "Alpha", Age: 30, Phone: "111", ProfilePic: "old.png"}
			if err := svc.CreatePatient(t.Context(), &seed); err != nil {
				t.Fatalf("seed patient: %v", err)
			}

			app := fiber.New()
			app.Patch("/patients/:id", func(c fiber.Ctx) error {
				c.Locals("current_user", user)
				return svc.HandleUpdate(c)
			})

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			for name, value := range tt.fields {
				if err := writer.WriteField(name, value); err != nil {
					t.Fatalf("write %s field: %v", name, err)
				}
			}
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

			req := httptest.NewRequest(http.MethodPatch, "/patients/"+seed.UID, body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			req.Header.Set("HX-Request", "true")
			resp, err := app.Test(req)
			wantStatus := fiber.StatusOK
			if tt.saved {
				wantStatus = fiber.StatusNoContent
			}
			if err != nil || resp.StatusCode != wantStatus {
				t.Fatalf("update patient expected %d, got status=%d err=%v", wantStatus, resp.StatusCode, err)
			}

			saved, err := svc.TakePatientByID(t.Context(), user.ID, seed.UID)
			if err != nil {
				t.Fatalf("reload patient: %v", err)
			}
			if tt.saved && (saved.ProfilePic == "" || saved.ProfilePic == "old.png") {
				t.Errorf("ProfilePic = %q, want the uploaded picture", saved.ProfilePic)
			}
			if !tt.saved && saved.ProfilePic != "old.png" {
				t.Errorf("ProfilePic = %q, want old.png kept after an invalid update", saved.ProfilePic)
			}
			entries, err := os.ReadDir(assetsDir)
			if err != nil {
				t.Fatalf("read assets dir: %v", err)
			}
			if !tt.saved && len(entries) != 0 {
				t.Errorf("assets dir has %d entries, want none after an invalid update", len(entries))
			}
		})
	}
}

func TestUpdatePatientByIDRejectsBlankRequiredFields(t *testing.T) {
	svc, user := newPatientServiceForTests(t)
	seed := patientUpsertInput{Name: "Original", Age: 30, Phone: "555"}.toPatient(0, "", user.ID)
	if err := svc.CreatePatient(t.Context(), &seed); err != nil {
		t.Fatalf("seed patient: %v", err)
	}

	cases := map[string]models.Patient{
		"blank name":  {Name: "", Age: 30, Phone: "555"},
		"zero age":    {Name: "Original", Age: 0, Phone: "555"},
		"blank phone": {Name: "Original", Age: 30, Phone: " "},
	}
	for name, updates := range cases {
		t.Run(name, func(t *testing.T) {
			err := svc.UpdatePatientByID(t.Context(), user.ID, seed.UID, updates)
			if !errors.Is(err, ErrInvalidPatient) {
				t.Fatalf("expected ErrInvalidPatient, got %v", err)
			}

			saved, err := svc.TakePatientByID(t.Context(), user.ID, seed.UID)
			if err != nil {
				t.Fatalf("reload patient: %v", err)
			}
			if saved.Name != "Original" || saved.Age != 30 || saved.Phone != "555" {
				t.Fatalf("patient changed by a rejected update: %+v", saved)
			}
		})
	}
}

func TestHandleUpdateRejectsBlankRequiredFields(t *testing.T) {
	svc, user := newPatientServiceForTests(t)
	seed := patientUpsertInput{Name: "Original", Age: 30, Phone: "555"}.toPatient(0, "", user.ID)
	if err := svc.CreatePatient(t.Context(), &seed); err != nil {
		t.Fatalf("seed patient: %v", err)
	}

	app := fiber.New()
	app.Patch("/patients/:id", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleUpdate(c)
	})
	form := url.Values{"name": {"Original Name"}, "age": {"30"}, "phone": {""}}
	req := httptest.NewRequest(http.MethodPatch, "/patients/"+seed.UID, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("update patient: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status=%d, want 200 with the form re-rendered", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	body := html.UnescapeString(string(raw))
	if !strings.Contains(body, "Name, age and phone can't be blank") {
		t.Fatalf("body has no blank-field toast")
	}
	if !strings.Contains(body, `value="Original Name"`) {
		t.Fatalf("body lost the submitted name")
	}
	if !strings.Contains(body, "hx-preserve") || !strings.Contains(body, "settle:0") {
		t.Fatalf("body does not keep the selected picture file: want hx-preserve on the input and settle:0 on the form swap")
	}
}

func TestAttachPatientPicRecordsAFailedSave(t *testing.T) {
	svc, user := newPatientServiceForTests(t)

	svc.ObjectStore = objstoretest.Failing{Err: errors.New("store down")}

	recorder := tracetest.NewSpanRecorder()
	svc.Tracer = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)).Tracer("test")

	patient := models.Patient{UserID: user.ID, Name: "Alpha", Age: 30, Phone: "111", ProfilePic: "old.png"}
	if err := svc.CreatePatient(t.Context(), &patient); err != nil {
		t.Fatalf("seed patient: %v", err)
	}

	var got string
	app := fiber.New()
	app.Post("/pic", func(c fiber.Ctx) error {
		got = svc.attachPatientPic(c, user.ID, patient)
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
	req := httptest.NewRequest(http.MethodPost, "/pic", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if _, err := app.Test(req); err != nil {
		t.Fatalf("attach picture: %v", err)
	}

	if got != "old.png" {
		t.Errorf("picture = %q, want the current one kept after a failed save", got)
	}
	spans := recorder.Ended()
	if len(spans) != 1 || spans[0].Name() != "patient/attachPic" {
		t.Fatalf("spans = %d, want one patient/attachPic span", len(spans))
	}
	if spans[0].Status().Code != codes.Error || len(spans[0].Events()) == 0 {
		t.Errorf("span status = %v with %d events, want an error with the exception recorded", spans[0].Status().Code, len(spans[0].Events()))
	}
}

func TestAttachPatientPicRecordsAFailedDBUpdate(t *testing.T) {
	svc, user := newPatientServiceForTests(t)
	useDiskStore(t, svc, t.TempDir())

	recorder := tracetest.NewSpanRecorder()
	svc.Tracer = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)).Tracer("test")

	patient := models.Patient{UserID: user.ID, Name: "Alpha", Age: 30, Phone: "111", ProfilePic: "old.png"}
	if err := svc.CreatePatient(t.Context(), &patient); err != nil {
		t.Fatalf("seed patient: %v", err)
	}

	var got string
	app := fiber.New()
	app.Post("/pic", func(c fiber.Ctx) error {
		got = svc.attachPatientPic(c, user.ID+1, patient)
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
		t.Fatalf("spans = %d, want one patient/attachPic span", len(spans))
	}
	if status := spans[0].Status(); status.Code != codes.Error || status.Description != "attach profile pic failed" {
		t.Errorf("span status = %v %q, want the attach failure", status.Code, status.Description)
	}
}

func TestAttachPatientPicDeletesTheReplacedFile(t *testing.T) {
	svc, user := newPatientServiceForTests(t)
	assetsDir := t.TempDir()
	useDiskStore(t, svc, assetsDir)

	patient := models.Patient{UserID: user.ID, Name: "Alpha", Age: 30, Phone: "111"}
	if err := svc.CreatePatient(t.Context(), &patient); err != nil {
		t.Fatalf("seed patient: %v", err)
	}

	oldName := patient.UID + "_ppic_old.png"
	if err := svc.ObjectStore.Put(t.Context(), ProfilePicKey(oldName), strings.NewReader("old"), 3, "image/png"); err != nil {
		t.Fatalf("write old picture: %v", err)
	}
	patient.ProfilePic = "/patients/" + patient.UID + "/profilepic/" + oldName

	var got string
	app := fiber.New()
	app.Post("/pic", func(c fiber.Ctx) error {
		got = svc.attachPatientPic(c, user.ID, patient)
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

	if got == "" || got == patient.ProfilePic {
		t.Fatalf("picture = %q, want the new one", got)
	}
	if _, err := svc.ObjectStore.Open(t.Context(), ProfilePicKey(oldName)); !errors.Is(err, objstore.ErrNotFound) {
		t.Errorf("old picture file still exists: %v", err)
	}
	dir := filepath.Join(assetsDir, "patients")
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Errorf("want only the new picture in %s, got %d entries err=%v", dir, len(entries), err)
	}
}

func TestHandleUpdateRejectsAnUnsupportedPicture(t *testing.T) {
	svc, user := newPatientServiceForTests(t)
	useDiskStore(t, svc, t.TempDir())

	seed := models.Patient{UserID: user.ID, Name: "Seed", Age: 30, Phone: "111"}
	if err := svc.CreatePatient(t.Context(), &seed); err != nil {
		t.Fatalf("seed patient: %v", err)
	}

	app := fiber.New()
	app.Patch("/patients/:id", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleUpdate(c)
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("name", "Renamed")
	_ = writer.WriteField("phone", "999")
	_ = writer.WriteField("age", "30")
	part, err := writer.CreateFormFile("profilePic", "scan.tiff")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	_, _ = part.Write([]byte("tiff"))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPatch, "/patients/"+seed.UID, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("HX-Request", "true")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status=%d err=%v, want 200 with the form re-rendered", resp.StatusCode, err)
	}
	raw, _ := io.ReadAll(resp.Body)
	page := html.UnescapeString(string(raw))
	if !strings.Contains(page, "Unsupported picture type") {
		t.Errorf("body has no unsupported-type toast")
	}
	if !strings.Contains(page, `value="999"`) {
		t.Errorf("body lost the submitted phone")
	}

	saved, err := svc.TakePatientByID(t.Context(), user.ID, seed.UID)
	if err != nil {
		t.Fatalf("reload patient: %v", err)
	}
	if saved.Name != "Seed" {
		t.Errorf("Name = %q, want the patient unchanged after a rejected upload", saved.Name)
	}
}
