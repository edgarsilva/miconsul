package patient

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"

	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
)

func TestHandleUpdateSavesClearedFields(t *testing.T) {
	svc, user := newPatientServiceForTests(t)

	var full patientUpsertInput
	fillFields(&full)
	filled := full.toPatient(0, "", user.ID)

	// The form does not render these fields, so an update must keep them.
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

	// A browser leaves unchecked checkboxes out of the form.
	form := url.Values{"name": {""}}
	req := httptest.NewRequest(http.MethodPatch, "/patients/"+seed.UID, strings.NewReader(form.Encode()))
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
	assertFieldsCleared(t, filled, saved, "UserID")
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

func TestHandleUpdateSavesUploadedProfilePic(t *testing.T) {
	svc, user := newPatientServiceForTests(t)
	svc.Env.AssetsDir = t.TempDir()

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
	if err := writer.WriteField("name", "Alpha"); err != nil {
		t.Fatalf("write name field: %v", err)
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
	if err != nil || resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("update patient expected 204, got status=%d err=%v", resp.StatusCode, err)
	}

	saved, err := svc.TakePatientByID(t.Context(), user.ID, seed.UID)
	if err != nil {
		t.Fatalf("reload patient: %v", err)
	}
	if saved.ProfilePic == "" || saved.ProfilePic == "old.png" {
		t.Errorf("ProfilePic = %q, want the uploaded picture", saved.ProfilePic)
	}
}
