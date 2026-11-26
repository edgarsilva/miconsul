package clinic

import (
	"bytes"
	"errors"
	"html"
	"io"
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
	svc, user := newClinicServiceForTests(t)

	var full clinicUpsertInput
	fillFields(&full)
	filled := full.toClinic(0, "", user.ID)

	notInForm := []string{"Line2", "Messenger"}
	seed := filled
	seed.ProfilePic = "old.png"
	for _, name := range notInForm {
		reflect.ValueOf(&seed).Elem().FieldByName(name).SetString("kept")
	}
	if err := svc.CreateClinic(t.Context(), &seed); err != nil {
		t.Fatalf("seed clinic: %v", err)
	}
	before, err := svc.TakeClinicByID(t.Context(), user.ID, seed.UID)
	if err != nil {
		t.Fatalf("load seeded clinic: %v", err)
	}

	app := fiber.New()
	app.Patch("/clinics/:id", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleUpdate(c)
	})

	requiredFieldsOnly := url.Values{"name": {"Renamed"}, "price": {"0"}}
	req := httptest.NewRequest(http.MethodPatch, "/clinics/"+seed.UID, strings.NewReader(requiredFieldsOnly.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != fiber.StatusOK {
		t.Fatalf("update clinic expected 200, got status=%d err=%v", resp.StatusCode, err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), "old.png") {
		t.Errorf("fragment does not show the existing profile picture when none is uploaded")
	}

	saved, err := svc.TakeClinicByID(t.Context(), user.ID, seed.UID)
	if err != nil {
		t.Fatalf("reload clinic: %v", err)
	}
	assertFieldsCleared(t, filled, saved, "UserID", "Name")
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

func TestHandleUpdateRejectsBlankName(t *testing.T) {
	svc, user := newClinicServiceForTests(t)
	seed := clinicUpsertInput{Name: "Original"}.toClinic(0, "", user.ID)
	if err := svc.CreateClinic(t.Context(), &seed); err != nil {
		t.Fatalf("seed clinic: %v", err)
	}

	err := svc.UpdateClinicByID(t.Context(), user.ID, seed.UID, models.Clinic{Name: "  "})
	if !errors.Is(err, ErrInvalidClinic) {
		t.Fatalf("expected ErrInvalidClinic, got %v", err)
	}

	saved, err := svc.TakeClinicByID(t.Context(), user.ID, seed.UID)
	if err != nil {
		t.Fatalf("reload clinic: %v", err)
	}
	if saved.Name != "Original" {
		t.Fatalf("Name = %q, want Original kept after a rejected update", saved.Name)
	}

	app := fiber.New()
	app.Patch("/clinics/:id", func(c fiber.Ctx) error {
		c.Locals("current_user", user)
		return svc.HandleUpdate(c)
	})
	req := httptest.NewRequest(http.MethodPatch, "/clinics/"+seed.UID, strings.NewReader(url.Values{"name": {""}, "phone": {"555-0100"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("update clinic: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status=%d, want 200 with the form re-rendered", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	body := html.UnescapeString(string(raw))
	if !strings.Contains(body, "Clinic name can't be blank") {
		t.Fatalf("body has no blank-name toast")
	}
	if !strings.Contains(body, `value="555-0100"`) {
		t.Fatalf("body lost the submitted phone")
	}
	if !strings.Contains(body, "hx-preserve") || !strings.Contains(body, "settle:0") {
		t.Fatalf("body does not keep the selected picture file: want hx-preserve on the input and settle:0 on the form swap")
	}
}

func TestHandleUpdateRejectsAnUnsupportedPicture(t *testing.T) {
	svc, user := newClinicServiceForTests(t)
	svc.Env.AssetsDir = t.TempDir()

	seed := models.Clinic{UserID: user.ID, Name: "Seed", Phone: "111"}
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
	_ = writer.WriteField("name", "Renamed")
	_ = writer.WriteField("phone", "999")
	_ = writer.WriteField("price", "5")
	part, err := writer.CreateFormFile("profilePic", "scan.tiff")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	_, _ = part.Write([]byte("tiff"))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPatch, "/clinics/"+seed.UID, body)
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

	saved, err := svc.TakeClinicByID(t.Context(), user.ID, seed.UID)
	if err != nil {
		t.Fatalf("reload clinic: %v", err)
	}
	if saved.Name != "Seed" {
		t.Errorf("Name = %q, want the clinic unchanged after a rejected upload", saved.Name)
	}
}
