package appointment

import (
	"fmt"
	"time"

	"github.com/edgarsilva/go-scaffold/internal/mailer"
	"github.com/edgarsilva/go-scaffold/internal/model"
	"github.com/edgarsilva/go-scaffold/internal/server"
)

type service struct {
	*server.Server
}

func NewService(s *server.Server) service {
	ser := service{
		Server: s,
	}

	ser.RegisterJobs()

	return ser
}

func (s *service) RegisterJobs() {
	_, err := s.BGJob.RunCronJob("0/1 * * * *", false, func() {
		appointments := []model.Appointment{}
		s.DB.
			Model(&model.Appointment{}).
			Preload("Patient").
			Preload("Clinic").
			Scopes(model.AppointmentWithPendingAlerts).
			Find(&appointments)
		for _, appointment := range appointments {
			s.SendReminderAlert(appointment)
		}
	})
	if err != nil {
		fmt.Println(err.Error())
	}
}

func (s *service) SendBookedAlert(appointment model.Appointment) error {
	return s.WP.Submit(func() {
		err := mailer.SendAppointmentBookedEmail(appointment)
		if err != nil {
			alert := model.Alert{
				Medium: model.AlertMediumEmail,
				Name:   "appointment_booked",
				Status: model.AlertFailed,
				To:     appointment.Patient.Email,
			}
			s.DB.Model(&appointment).Association("Alerts").Append(&alert)
			return
		}

		s.DB.Model(&appointment).Update("BookedAlertSentAt", time.Now())

		alert := model.Alert{
			Medium: model.AlertMediumEmail,
			Name:   "appointment_booked",
			Status: model.AlertSent,
			To:     appointment.Patient.Email,
		}
		s.DB.Model(&appointment).Association("Alerts").Append(&alert)
	})
}

func (s *service) SendReminderAlert(appointment model.Appointment) error {
	return s.WP.Submit(func() {
		err := mailer.SendAppointmentReminderEmail(appointment)
		if err != nil {
			alert := model.Alert{
				Medium: model.AlertMediumEmail,
				Name:   "appointment_reminder",
				Status: model.AlertFailed,
				To:     appointment.Patient.Email,
			}
			s.DB.Model(&appointment).Association("Alerts").Append(&alert)
			return
		}

		s.DB.Model(&appointment).Update("reminder_alert_sent_at", time.Now())

		alert := model.Alert{
			Medium: model.AlertMediumEmail,
			Name:   "appointment_reminder",
			Status: model.AlertSent,
			To:     appointment.Patient.Email,
		}
		s.DB.Model(&appointment).Association("Alerts").Append(&alert)
	})
}
