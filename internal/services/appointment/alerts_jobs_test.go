package appointment

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"miconsul/internal/jobs"
	"miconsul/internal/models"

	"gorm.io/gorm"
)

func TestSendReminderAlertExecutesWorkerPath(t *testing.T) {
	svc, user, clinic, patient := newAppointmentServiceForTests(t)

	apnt := models.Appointment{
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

	apnt := models.Appointment{
		UserID:    user.ID,
		ClinicID:  clinic.ID,
		PatientID: patient.ID,
		BookedAt:  time.Now().Add(time.Hour),
		Token:     "tok_booked",
		Patient:   patient,
	}
	if err := svc.CreateAppointment(t.Context(), &apnt); err != nil {
		t.Fatalf("create appointment: %v", err)
	}

	notification := models.Notification{
		Medium: models.NotificationMediumEmail,
		Name:   "appointment_booked",
		Status: models.NotificationSent,
		To:     patient.Email,
	}
	if err := svc.DB.WithContext(t.Context()).Model(&apnt).Association("Notifications").Append(&notification); err != nil {
		t.Fatalf("append booked notification: %v", err)
	}

	payload, err := json.Marshal(TaskAppointmentPayload{AppointmentID: apnt.UID})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	if err := svc.handleBookedAlertTask(context.Background(), jobs.Task{Payload: payload}); err != nil {
		t.Fatalf("expected idempotent skip, got %v", err)
	}
}

func TestAppendNotificationPersistsPrimaryKeyAlertableID(t *testing.T) {
	svc, user, clinic, patient := newAppointmentServiceForTests(t)

	apnt := models.Appointment{
		UserID:    user.ID,
		ClinicID:  clinic.ID,
		PatientID: patient.ID,
		BookedAt:  time.Now().Add(time.Hour),
		Token:     "tok_uid_persist",
		Patient:   patient,
	}
	if err := svc.CreateAppointment(t.Context(), &apnt); err != nil {
		t.Fatalf("create appointment: %v", err)
	}

	svc.appendNotification(t.Context(), apnt, models.Notification{
		Medium: models.NotificationMediumEmail,
		Name:   "appointment_booked",
		Status: models.NotificationSent,
		To:     patient.Email,
	})

	notification, err := gorm.G[models.Notification](svc.DB.GormDB()).
		Where("name = ?", "appointment_booked").
		Where("`to` = ?", patient.Email).
		Order("id desc").
		First(t.Context())
	if err != nil {
		t.Fatalf("load notification: %v", err)
	}

	if notification.NotificationableType != "appointments" {
		t.Fatalf("expected alertable_type appointments, got %q", notification.NotificationableType)
	}
	wantID := strconv.FormatUint(uint64(apnt.ID), 10)
	if notification.NotificationableID != wantID {
		t.Fatalf("expected alertable_id %q, got %q", wantID, notification.NotificationableID)
	}
}

func TestNotificationSentAnyMediumUsesPrimaryKeyAlertableID(t *testing.T) {
	svc, user, clinic, patient := newAppointmentServiceForTests(t)

	apnt := models.Appointment{
		UserID:    user.ID,
		ClinicID:  clinic.ID,
		PatientID: patient.ID,
		BookedAt:  time.Now().Add(time.Hour),
		Token:     "tok_legacy_numeric",
		Patient:   patient,
	}
	if err := svc.CreateAppointment(t.Context(), &apnt); err != nil {
		t.Fatalf("create appointment: %v", err)
	}

	legacy := models.Notification{
		Medium:               models.NotificationMediumEmail,
		Name:                 "appointment_reminder",
		Status:               models.NotificationSent,
		To:                   patient.Email,
		NotificationableID:   strconv.FormatUint(uint64(apnt.ID), 10),
		NotificationableType: "appointments",
	}
	if err := svc.DB.WithContext(t.Context()).Create(&legacy).Error; err != nil {
		t.Fatalf("create legacy notification: %v", err)
	}

	sent, err := svc.notificationSentAnyMedium(t.Context(), apnt, "appointment_reminder")
	if err != nil {
		t.Fatalf("notificationSentAnyMedium returned error: %v", err)
	}
	if !sent {
		t.Fatalf("expected reminder to be treated as already sent")
	}
}
