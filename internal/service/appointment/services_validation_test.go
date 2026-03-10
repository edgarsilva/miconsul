package appointment

import (
	"context"
	"errors"
	"testing"

	"miconsul/internal/model"
)

func TestServiceInputValidation(t *testing.T) {
	svc := &service{}
	ctx := context.Background()

	t.Run("create appointment validates required input", func(t *testing.T) {
		if err := svc.CreateAppointment(ctx, nil); err == nil {
			t.Fatalf("expected nil appointment error")
		}

		apnt := &model.Appointment{Status: model.AppointmentStatus("nope")}
		if err := svc.CreateAppointment(ctx, apnt); err == nil {
			t.Fatalf("expected invalid status error")
		}
	})

	t.Run("id-guarded queries fail fast on blank id", func(t *testing.T) {
		if _, err := svc.TakePatientByID(ctx, "usr_1", "   "); !errors.Is(err, ErrIDRequired) {
			t.Fatalf("expected ErrIDRequired for patient id, got %v", err)
		}
		if _, err := svc.TakeClinicByID(ctx, "usr_1", ""); !errors.Is(err, ErrIDRequired) {
			t.Fatalf("expected ErrIDRequired for clinic id, got %v", err)
		}
		if _, err := svc.TakeAppointmentByID(ctx, "usr_1", ""); !errors.Is(err, ErrIDRequired) {
			t.Fatalf("expected ErrIDRequired for appointment id, got %v", err)
		}
		if _, err := svc.TakePatientByIDWithLastDoneAppointment(ctx, "usr_1", ""); !errors.Is(err, ErrIDRequired) {
			t.Fatalf("expected ErrIDRequired for patient id with preload, got %v", err)
		}
	})

	t.Run("id-guarded mutations fail fast on blank id", func(t *testing.T) {
		if err := svc.UpdateAppointmentByID(ctx, "usr_1", "", appointmentPatchUpdates{}); !errors.Is(err, ErrIDRequired) {
			t.Fatalf("expected ErrIDRequired for update, got %v", err)
		}
		if err := svc.DeleteAppointmentByID(ctx, "usr_1", ""); !errors.Is(err, ErrIDRequired) {
			t.Fatalf("expected ErrIDRequired for delete, got %v", err)
		}
		if err := svc.CompleteAppointmentByID(ctx, "usr_1", "", appointmentCompleteUpdates{}); !errors.Is(err, ErrIDRequired) {
			t.Fatalf("expected ErrIDRequired for complete, got %v", err)
		}
		if err := svc.CancelAppointmentByID(ctx, "usr_1", "", appointmentCancelUpdates{}); !errors.Is(err, ErrIDRequired) {
			t.Fatalf("expected ErrIDRequired for cancel, got %v", err)
		}
	})

	t.Run("status validation catches invalid mutations", func(t *testing.T) {
		if err := svc.CompleteAppointmentByID(ctx, "usr_1", "apnt_1", appointmentCompleteUpdates{Status: model.AppointmentStatus("invalid")}); err == nil {
			t.Fatalf("expected complete invalid status error")
		}
		if err := svc.CancelAppointmentByID(ctx, "usr_1", "apnt_1", appointmentCancelUpdates{Status: model.AppointmentStatus("invalid")}); err == nil {
			t.Fatalf("expected cancel invalid status error")
		}
		if err := svc.UpdateAppointmentByIDAndToken(ctx, "apnt_1", "tok", []string{"Status"}, model.Appointment{Status: model.AppointmentStatus("invalid")}); err == nil {
			t.Fatalf("expected token update invalid status error")
		}
	})

	t.Run("update columns validates required columns", func(t *testing.T) {
		err := svc.updateAppointmentColumnsByID(ctx, "usr_1", "apnt_1", nil, appointmentPatchUpdates{})
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
