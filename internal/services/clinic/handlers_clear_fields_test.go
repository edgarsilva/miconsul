package clinic

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestHandleUpdateSavesClearedFields(t *testing.T) {
	svc, user := newClinicServiceForTests(t)

	var full clinicUpsertInput
	fillFields(&full)
	filled := full.toClinic(0, "", user.ID, 50000)

	// The form does not render these fields, so an update must keep them.
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

	form := url.Values{"name": {""}, "price": {"0"}}
	req := httptest.NewRequest(http.MethodPatch, "/clinics/"+seed.UID, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != fiber.StatusOK {
		t.Fatalf("update clinic expected 200, got status=%d err=%v", resp.StatusCode, err)
	}

	saved, err := svc.TakeClinicByID(t.Context(), user.ID, seed.UID)
	if err != nil {
		t.Fatalf("reload clinic: %v", err)
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
