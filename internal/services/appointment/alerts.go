package appointment

import (
	"context"
	"fmt"
	"time"

	"miconsul/internal/mailer"
	"miconsul/internal/models"

	"gorm.io/gorm"
)

func (s *service) DispatchBookedAlert(appointment models.Appointment) error {
	if s.Env.JobsEnabled {
		payload := TaskAppointmentPayload{AppointmentID: appointment.UID}
		_, err := s.EnqueueTask(context.Background(), TaskBookedAlert, payload)
		return err
	}

	return s.SendToWorker(context.Background(), func() {
		ctx, cancel := context.WithTimeout(context.Background(), defaultWorkerContextTimeout)
		defer cancel()

		s.sendBookedNow(ctx, appointment)
	})
}

func (s *service) SendReminder(appointment models.Appointment) error {
	return s.SendToWorker(context.Background(), func() {
		ctx, cancel := context.WithTimeout(context.Background(), defaultWorkerContextTimeout)
		defer cancel()

		s.sendReminderNow(ctx, appointment)
	})
}

func (s *service) SendReminderAlert(appointment models.Appointment) error {
	return s.SendReminder(appointment)
}

func (s *service) sendReminderNow(ctx context.Context, appointment models.Appointment) {
	err := mailer.SendAppointmentReminderEmail(s.Env, appointment)
	if err != nil {
		notification := models.Notification{
			Medium: models.NotificationMediumEmail,
			Name:   "appointment_reminder",
			Status: models.NotificationFailed,
			To:     appointment.Patient.Email,
		}
		appendErr := s.DB.WithContext(ctx).Model(&appointment).Association("Notifications").Append(&notification)
		if appendErr != nil {
			fmt.Println("failed to append failed reminder notification:", appendErr.Error())
		}
		return
	}

	_, updateErr := gorm.G[models.Appointment](s.DB.GormDB()).
		Where("uid = ?", appointment.UID).
		Update(ctx, "ReminderAlertSentAt", time.Now())
	if updateErr != nil {
		fmt.Println("failed to update ReminderAlertSentAt:", updateErr.Error())
	}

	notification := models.Notification{
		Medium: models.NotificationMediumEmail,
		Name:   "appointment_reminder",
		Status: models.NotificationSent,
		To:     appointment.Patient.Email,
	}
	appendErr := s.DB.WithContext(ctx).Model(&appointment).Association("Notifications").Append(&notification)
	if appendErr != nil {
		fmt.Println("failed to append sent reminder notification:", appendErr.Error())
	}
}

func (s *service) sendBookedNow(ctx context.Context, appointment models.Appointment) {
	notification := models.Notification{
		Medium: models.NotificationMediumEmail,
		Name:   "appointment_booked",
		Status: models.NotificationFailed,
		To:     appointment.Patient.Email,
	}
	err := mailer.SendAppointmentBookedEmail(s.Env, appointment)
	if err != nil {
		appendErr := s.DB.WithContext(ctx).Model(&appointment).Association("Notifications").Append(&notification)
		if appendErr != nil {
			fmt.Println("failed to append failed booked notification:", appendErr.Error())
		}
		return
	}
	_, updateErr := gorm.G[models.Appointment](s.DB.GormDB()).
		Where("uid = ?", appointment.UID).
		Update(ctx, "BookedAlertSentAt", time.Now())
	if updateErr != nil {
		fmt.Println("failed to update BookedAlertSentAt:", updateErr.Error())
	}

	notification.Status = models.NotificationSent
	appendErr := s.DB.WithContext(ctx).Model(&appointment).Association("Notifications").Append(&notification)
	if appendErr != nil {
		fmt.Println("failed to append sent booked notification:", appendErr.Error())
	}
}
