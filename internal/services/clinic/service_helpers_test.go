package clinic

import (
	"context"
	"testing"

	"miconsul/internal/models"
)

func TestServiceIDGuards(t *testing.T) {
	svc := service{}
	ctx := context.Background()

	if _, err := svc.TakeClinicByID(ctx, 1, ""); err != ErrIDRequired {
		t.Fatalf("expected ErrIDRequired from TakeClinicByID, got %v", err)
	}
	if err := svc.UpdateClinicByID(ctx, 1, "", models.Clinic{}); err != ErrIDRequired {
		t.Fatalf("expected ErrIDRequired from UpdateClinicByID, got %v", err)
	}
	if err := svc.DeleteClinicByID(ctx, 1, ""); err != ErrIDRequired {
		t.Fatalf("expected ErrIDRequired from DeleteClinicByID, got %v", err)
	}
	if _, err := svc.clinicExistsByID(ctx, 1, ""); err != ErrIDRequired {
		t.Fatalf("expected ErrIDRequired from clinicExistsByID, got %v", err)
	}
}
