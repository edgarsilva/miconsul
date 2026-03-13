package appointment

import (
	"strings"
	"testing"
	"time"

	"miconsul/internal/model"
)

func TestSendReminderAlertExecutesWorkerPath(t *testing.T) {
	svc, user, clinic, patient := newAppointmentServiceForTests(t)

	apnt := model.Appointment{
		UserID:    user.ID,
		ClinicID:  clinic.ID,
		PatientID: patient.ID,
		BookedAt:  time.Now().Add(time.Hour),
		Token:     "tok_reminder",
		Patient:   patient,
	}
	if err := svc.CreateAppointment(t.Context(), &apnt); err != nil {
		t.Fatalf("create appointment: %v", err)
	}

	if err := svc.SendReminderAlert(apnt); err != nil {
		t.Fatalf("expected reminder worker enqueue success, got %v", err)
	}
}

func TestRegisterCronJobReturnsWrappedErrorWithoutCronRunner(t *testing.T) {
	svc, _, _, _ := newAppointmentServiceForTests(t)

	err := svc.RegisterCronJob()
	if err == nil {
		t.Fatalf("expected register cron job error when cron runner is nil")
	}
	if !strings.Contains(err.Error(), "failed to register appointment reminder cron job") {
		t.Fatalf("expected wrapped register cron error, got %v", err)
	}
}
