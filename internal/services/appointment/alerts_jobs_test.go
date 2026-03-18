package appointment

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"miconsul/internal/jobs"
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

func TestHandleBookedAlertTaskSkipsWhenAlreadySent(t *testing.T) {
	svc, user, clinic, patient := newAppointmentServiceForTests(t)

	apnt := model.Appointment{
		UserID:            user.ID,
		ClinicID:          clinic.ID,
		PatientID:         patient.ID,
		BookedAt:          time.Now().Add(time.Hour),
		Token:             "tok_booked",
		Patient:           patient,
		BookedAlertSentAt: time.Now(),
	}
	if err := svc.CreateAppointment(t.Context(), &apnt); err != nil {
		t.Fatalf("create appointment: %v", err)
	}

	payload, err := json.Marshal(TaskAppointmentPayload{AppointmentID: apnt.ID})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	if err := svc.handleBookedAlertTask(context.Background(), jobs.Task{Payload: payload}); err != nil {
		t.Fatalf("expected idempotent skip, got %v", err)
	}
}
