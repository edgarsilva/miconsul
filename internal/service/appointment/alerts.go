package appointment

import (
	"fmt"
	"time"

	"miconsul/internal/mailer"
	"miconsul/internal/model"

	"gorm.io/gorm"
)

func (s *service) SendBookedAlert(appointment model.Appointment) error {
	err := s.SendToWorker(func() {
		ctx, cancel := s.newWorkerContext()
		defer cancel()

		err := mailer.SendAppointmentBookedEmail(appointment)
		if err != nil {
			alert := model.Alert{
				Medium: model.AlertMediumEmail,
				Name:   "appointment_booked",
				Status: model.AlertFailed,
				To:     appointment.Patient.Email,
			}
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

		alert := model.Alert{
			Medium: model.AlertMediumEmail,
			Name:   "appointment_booked",
			Status: model.AlertSent,
			To:     appointment.Patient.Email,
		}
		appendErr := s.DB.WithContext(ctx).Model(&appointment).Association("Alerts").Append(&alert)
		if appendErr != nil {
			fmt.Println("failed to append sent booked alert:", appendErr.Error())
		}
	})

	return err
}

func (s *service) SendReminderAlert(appointment model.Appointment) error {
	err := s.SendToWorker(func() {
		ctx, cancel := s.newWorkerContext()
		defer cancel()

		err := mailer.SendAppointmentReminderEmail(appointment)
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
	})

	return err
}
