package patient

import (
	"strings"
	"testing"

	"miconsul/internal/models"
)

func TestNormalizePatientWriteInput(t *testing.T) {
	t.Run("nil patient returns error", func(t *testing.T) {
		if err := normalizePatientWriteInput(nil); err == nil {
			t.Fatalf("expected nil patient to fail")
		}
	})

	t.Run("trims and normalizes fields", func(t *testing.T) {
		patient := &models.Patient{
			Name:  "  Patient Name  ",
			Email: "  MIXED@Example.COM  ",
			Phone: "  +12345  ",
		}

		if err := normalizePatientWriteInput(patient); err != nil {
			t.Fatalf("expected normalized input to pass: %v", err)
		}
		if patient.Name != "Patient Name" {
			t.Fatalf("expected trimmed name, got %q", patient.Name)
		}
		if patient.Email != "mixed@example.com" {
			t.Fatalf("expected normalized email, got %q", patient.Email)
		}
		if patient.Phone != "+12345" {
			t.Fatalf("expected trimmed phone, got %q", patient.Phone)
		}
	})

	t.Run("rejects over max lengths", func(t *testing.T) {
		cases := []struct {
			name    string
			patient models.Patient
		}{
			{name: "name too long", patient: models.Patient{Name: strings.Repeat("n", 121)}},
			{name: "email too long", patient: models.Patient{Email: strings.Repeat("e", 255)}},
			{name: "phone too long", patient: models.Patient{Phone: strings.Repeat("p", 41)}},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				p := tc.patient
				if err := normalizePatientWriteInput(&p); err == nil {
					t.Fatalf("expected boundary validation error")
				}
			})
		}
	})
}
