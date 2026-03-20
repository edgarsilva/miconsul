package appointment

import (
	"context"
	"fmt"
	"time"

	"miconsul/internal/mailer"
	"miconsul/internal/models"

	"gorm.io/gorm"
)

func (s *service) DispatchBookedAlert(appointment model.Appointment) error {
	if s.Env.JobsEnabled {
		payload := TaskAppointmentPayload{AppointmentID: appointment.ID}
		_, err := s.EnqueueTask(context.Background(), TaskBookedAlert, payload)
		return err
	}

	return s.SendToWorker(func() {
		ctx, cancel := context.WithTimeout(context.Background(), defaultWorkerContextTimeout)
		defer cancel()

		s.sendBookedNow(ctx, appointment)
	})
}

func (s *service) SendReminder(appointment model.Appointment) error {
	err := s.SendToWorker(func() {
		ctx, cancel := context.WithTimeout(context.Background(), defaultWorkerContextTimeout)
		defer cancel()

		s.sendReminderNow(ctx, appointment)
	})

	return err
}

func (s *service) SendReminderAlert(appointment model.Appointment) error {
	return s.SendReminder(appointment)
}

func (s *service) sendReminderNow(ctx context.Context, appointment model.Appointment) {
	err := mailer.SendAppointmentReminderEmail(s.Env, appointment)
	if err != nil {
		alert := model.Alert{
			Medium: model.AlertMediumEmail,
			Name:   "appointment_reminder",
			Status: model.AlertFailed,
			To:     appointment.Patient.Email,
		}
		appendErr := s.DB.WithContext(ctx).Model(&appointment).Association("Alerts").Append(&alert)
		if appendErr != nil {
			fmt.Println("failed to append failed reminder alert:", appendErr.Error())
		}
		return
	}

	_, updateErr := gorm.G[model.Appointment](s.DB.GormDB()).
		Where("id = ?", appointment.ID).
		Update(ctx, "ReminderAlertSentAt", time.Now())
	if updateErr != nil {
		fmt.Println("failed to update ReminderAlertSentAt:", updateErr.Error())
	}

	alert := model.Alert{
		Medium: model.AlertMediumEmail,
		Name:   "appointment_reminder",
		Status: model.AlertSent,
		To:     appointment.Patient.Email,
	}
	appendErr := s.DB.WithContext(ctx).Model(&appointment).Association("Alerts").Append(&alert)
	if appendErr != nil {
		fmt.Println("failed to append sent reminder alert:", appendErr.Error())
	}
}

func (s *service) sendBookedNow(ctx context.Context, appointment model.Appointment) {
	alert := model.Alert{
		Medium: model.AlertMediumEmail,
		Name:   "appointment_booked",
		Status: model.AlertFailed,
		To:     appointment.Patient.Email,
	}
	err := mailer.SendAppointmentBookedEmail(s.Env, appointment)
	if err != nil {
		appendErr := s.DB.WithContext(ctx).Model(&appointment).Association("Alerts").Append(&alert)
		if appendErr != nil {
			fmt.Println("failed to append failed booked alert:", appendErr.Error())
		}
		return
	}
	_, updateErr := gorm.G[model.Appointment](s.DB.GormDB()).
		Where("id = ?", appointment.ID).
		Update(ctx, "BookedAlertSentAt", time.Now())
	if updateErr != nil {
		fmt.Println("failed to update BookedAlertSentAt:", updateErr.Error())
	}

	alert.Status = model.AlertSent
	appendErr := s.DB.WithContext(ctx).Model(&appointment).Association("Alerts").Append(&alert)
	if appendErr != nil {
		fmt.Println("failed to append sent booked alert:", appendErr.Error())
	}
}
