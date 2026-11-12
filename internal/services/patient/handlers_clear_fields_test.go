package patient

import (
	"bytes"
	"encoding/hex"
	"errors"
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
	// Name, age and phone are required, so they stay; every other field is cleared.
	form := url.Values{"name": {"Renamed"}, "age": {"1"}, "phone": {"1"}}
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
	form := url.Values{"name": {"Original"}, "age": {"30"}, "phone": {""}}
	req := httptest.NewRequest(http.MethodPatch, "/patients/"+seed.UID, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("update patient: %v", err)
	}
	if resp.StatusCode != fiber.StatusNoContent || resp.Header.Get("HX-Redirect") != "/patients/"+seed.UID {
		t.Fatalf("status=%d HX-Redirect=%q, want 204 back to the patient", resp.StatusCode, resp.Header.Get("HX-Redirect"))
	}
	if flash := flashText(t, resp); !strings.Contains(flash, "Name, age and phone can't be blank") {
		t.Fatalf("flash = %q, want the blank-field message", flash)
	}
}

// flashText returns the flash cookie's decoded content, which carries the toast message.
func flashText(t *testing.T, resp *http.Response) string {
	t.Helper()

	for _, cookie := range resp.Cookies() {
		if cookie.Name != "fiber_flash" {
			continue
		}
		raw, err := hex.DecodeString(cookie.Value)
		if err != nil {
			t.Fatalf("decode flash cookie: %v", err)
		}
		return string(raw)
	}

	return ""
}
