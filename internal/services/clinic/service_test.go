package clinic

import (
	"strings"
	"testing"

	"miconsul/internal/models"
)

func TestNormalizeClinicWriteInput(t *testing.T) {
	t.Run("nil clinic returns error", func(t *testing.T) {
		if err := normalizeClinicWriteInput(nil); err == nil {
			t.Fatalf("expected nil clinic to fail")
		}
	})

	t.Run("trims and normalizes fields", func(t *testing.T) {
		clinic := &models.Clinic{
			Name:  "  Clinic Name  ",
			Email: "  MIXED@Example.COM  ",
			Phone: "  +12345  ",
		}

		if err := normalizeClinicWriteInput(clinic); err != nil {
			t.Fatalf("expected normalized input to pass: %v", err)
		}
		if clinic.Name != "Clinic Name" {
			t.Fatalf("expected trimmed name, got %q", clinic.Name)
		}
		if clinic.Email != "mixed@example.com" {
			t.Fatalf("expected normalized email, got %q", clinic.Email)
		}
		if clinic.Phone != "+12345" {
			t.Fatalf("expected trimmed phone, got %q", clinic.Phone)
		}
	})

	t.Run("rejects over max lengths", func(t *testing.T) {
		cases := []struct {
			name   string
			clinic models.Clinic
		}{
			{name: "name too long", clinic: models.Clinic{Name: strings.Repeat("n", 121)}},
			{name: "email too long", clinic: models.Clinic{Email: strings.Repeat("e", 255)}},
			{name: "phone too long", clinic: models.Clinic{Phone: strings.Repeat("p", 41)}},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				c := tc.clinic
				if err := normalizeClinicWriteInput(&c); err == nil {
					t.Fatalf("expected boundary validation error")
				}
			})
		}
	})
}

func TestNewServiceNilServer(t *testing.T) {
	_, err := NewService(nil)
	if err == nil {
		t.Fatalf("expected nil server to fail")
	}
}
