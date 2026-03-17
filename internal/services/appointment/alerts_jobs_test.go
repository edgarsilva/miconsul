package appointment

import (
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
