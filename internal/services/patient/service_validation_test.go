package patient

import (
	"context"
	"errors"
	"testing"

	"miconsul/internal/models"
)

func TestServiceValidationGuards(t *testing.T) {
	svc := service{}
	ctx := context.Background()

	t.Run("id-required methods return ErrIDRequired on blank id", func(t *testing.T) {
		if _, err := svc.TakePatientByID(ctx, "usr_1", "   "); !errors.Is(err, ErrIDRequired) {
			t.Fatalf("expected ErrIDRequired for TakePatientByID, got %v", err)
		}
		if err := svc.UpdatePatientByID(ctx, "usr_1", "", models.Patient{}); !errors.Is(err, ErrIDRequired) {
			t.Fatalf("expected ErrIDRequired for UpdatePatientByID, got %v", err)
		}
		if err := svc.DeletePatientByID(ctx, "usr_1", ""); !errors.Is(err, ErrIDRequired) {
			t.Fatalf("expected ErrIDRequired for DeletePatientByID, got %v", err)
		}
		if err := svc.ClearPatientProfilePic(ctx, "usr_1", ""); !errors.Is(err, ErrIDRequired) {
			t.Fatalf("expected ErrIDRequired for ClearPatientProfilePic, got %v", err)
		}
		if _, err := svc.patientExistsByID(ctx, "usr_1", ""); !errors.Is(err, ErrIDRequired) {
			t.Fatalf("expected ErrIDRequired for patientExistsByID, got %v", err)
		}
	})

	t.Run("PatientForShowPage short-circuits for create flow", func(t *testing.T) {
		patient, err := svc.PatientForShowPage(ctx, "usr_1", "new")
		if err != nil {
			t.Fatalf("expected nil error for new patient page, got %v", err)
		}
		if patient.ID != "" {
			t.Fatalf("expected zero patient for new page, got %+v", patient)
		}
	})
}

func TestNormalizeSearchTerm(t *testing.T) {
	t.Run("rejects short non-empty search term", func(t *testing.T) {
		if _, err := normalizeSearchTerm("ab"); err == nil {
			t.Fatalf("expected short term validation error")
		}
	})

	t.Run("accepts empty and trimmed term", func(t *testing.T) {
		term, err := normalizeSearchTerm("   abc   ")
		if err != nil {
			t.Fatalf("expected valid search term, got %v", err)
		}
		if term != "abc" {
			t.Fatalf("expected trimmed term, got %q", term)
		}
	})
}

func TestNewServiceNilServer(t *testing.T) {
	if _, err := NewService(nil); err == nil {
		t.Fatalf("expected nil server error")
	}
}
