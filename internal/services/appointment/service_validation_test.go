package appointment

import (
	"context"
	"errors"
	"testing"

	"miconsul/internal/models"
)

func TestServiceInputValidation(t *testing.T) {
	svc := &service{}
	ctx := context.Background()

	t.Run("create appointment validates required input", func(t *testing.T) {
		if err := svc.CreateAppointment(ctx, nil); err == nil {
			t.Fatalf("expected nil appointment error")
		}

		apnt := &models.Appointment{Status: models.AppointmentStatus("nope")}
		if err := svc.CreateAppointment(ctx, apnt); err == nil {
			t.Fatalf("expected invalid status error")
		}
	})

	t.Run("id-guarded queries fail fast on blank id", func(t *testing.T) {
		if _, err := svc.TakePatientByUID(ctx, 1, "   "); !errors.Is(err, ErrIDRequired) {
			t.Fatalf("expected ErrIDRequired for patient id, got %v", err)
		}
		if _, err := svc.TakeClinicByUID(ctx, 1, ""); !errors.Is(err, ErrIDRequired) {
			t.Fatalf("expected ErrIDRequired for clinic id, got %v", err)
		}
		if _, err := svc.TakeAppointmentByUID(ctx, 1, ""); !errors.Is(err, ErrIDRequired) {
			t.Fatalf("expected ErrIDRequired for appointment id, got %v", err)
		}
		if _, err := svc.TakePatientByUIDWithLastDoneAppointment(ctx, 1, ""); !errors.Is(err, ErrIDRequired) {
			t.Fatalf("expected ErrIDRequired for patient id with preload, got %v", err)
		}
	})

	t.Run("id-guarded mutations fail fast on blank id", func(t *testing.T) {
		if err := svc.UpdateAppointmentByID(ctx, 1, "", appointmentPatchUpdates{}); !errors.Is(err, ErrIDRequired) {
			t.Fatalf("expected ErrIDRequired for update, got %v", err)
		}
		if err := svc.DeleteAppointmentByID(ctx, 1, ""); !errors.Is(err, ErrIDRequired) {
			t.Fatalf("expected ErrIDRequired for delete, got %v", err)
		}
		if err := svc.CompleteAppointmentByID(ctx, 1, "", appointmentCompleteUpdates{}); !errors.Is(err, ErrIDRequired) {
			t.Fatalf("expected ErrIDRequired for complete, got %v", err)
		}
		if err := svc.CancelAppointmentByID(ctx, 1, "", appointmentCancelUpdates{}); !errors.Is(err, ErrIDRequired) {
			t.Fatalf("expected ErrIDRequired for cancel, got %v", err)
		}
	})

	t.Run("token-guarded lookups and mutations fail fast on blank token", func(t *testing.T) {
		for _, token := range []string{"", "   "} {
			if _, err := svc.TakeAppointmentByToken(ctx, token); !errors.Is(err, ErrTokenRequired) {
				t.Fatalf("take with token %q: expected ErrTokenRequired, got %v", token, err)
			}
			if err := svc.UpdateAppointmentByToken(ctx, token, []string{"Status"}, nil, appointmentTokenUpdates{}); !errors.Is(err, ErrTokenRequired) {
				t.Fatalf("update with token %q: expected ErrTokenRequired, got %v", token, err)
			}
			if err := svc.ConfirmAppointmentByToken(ctx, token); !errors.Is(err, ErrTokenRequired) {
				t.Fatalf("confirm with token %q: expected ErrTokenRequired, got %v", token, err)
			}
			if err := svc.CancelAppointmentByToken(ctx, token); !errors.Is(err, ErrTokenRequired) {
				t.Fatalf("cancel with token %q: expected ErrTokenRequired, got %v", token, err)
			}
			if err := svc.RequestAppointmentDateChangeByToken(ctx, token); !errors.Is(err, ErrTokenRequired) {
				t.Fatalf("date change with token %q: expected ErrTokenRequired, got %v", token, err)
			}
		}
	})

	t.Run("status validation catches invalid mutations", func(t *testing.T) {
		if err := svc.CompleteAppointmentByID(ctx, 1, "apnt_1", appointmentCompleteUpdates{Status: models.AppointmentStatus("invalid")}); err == nil {
			t.Fatalf("expected complete invalid status error")
		}
		if err := svc.CancelAppointmentByID(ctx, 1, "apnt_1", appointmentCancelUpdates{Status: models.AppointmentStatus("invalid")}); err == nil {
			t.Fatalf("expected cancel invalid status error")
		}
		if err := svc.UpdateAppointmentByToken(ctx, "tok", []string{"Status"}, nil, appointmentTokenUpdates{Status: models.AppointmentStatus("invalid")}); err == nil {
			t.Fatalf("expected token update invalid status error")
		}
	})

	t.Run("update columns validates required columns", func(t *testing.T) {
		err := svc.updateAppointmentColumnsByID(ctx, 1, "apnt_1", nil, appointmentPatchUpdates{})
		if err == nil {
			t.Fatalf("expected columns required error")
		}
	})
}

func TestNewServiceNilServer(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Fatalf("expected nil server error")
	}
}
