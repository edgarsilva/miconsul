package appointment

import (
	"errors"
	"fmt"
	"time"

	"miconsul/internal/common"
	"miconsul/internal/mailer"
	"miconsul/internal/model"
	"miconsul/internal/server"
	"github.com/gofiber/fiber/v2"
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

func (s *service) GetPatientByID(c *fiber.Ctx, id string) (model.Patient, error) {
	if id == "" {
		return model.Patient{}, nil
	}

	cu, _ := s.CurrentUser(c)
	patient := model.Patient{ID: id, UserID: cu.ID}
	result := s.DB.Model(&patient).Take(&patient)
	if result.RowsAffected != 1 {
		return model.Patient{}, errors.New("incorrect number of patient rows, expecting: 1, got:" + string(result.RowsAffected))
	}

	return patient, nil
}

func (s *service) GetClinicByID(c *fiber.Ctx, id string) (model.Clinic, error) {
	if id == "" {
		return model.Clinic{}, nil
	}

	cu, _ := s.CurrentUser(c)
	clinic := model.Clinic{ID: id, UserID: cu.ID}
	result := s.DB.Model(&clinic).Take(&clinic)
	if result.RowsAffected != 1 {
		return model.Clinic{}, errors.New("incorrect number of clinic rows, expecting: 1, got:" + string(result.RowsAffected))
	}

	return clinic, nil
}

func (s *service) GetAppointmentsBy(c *fiber.Ctx, cu model.User, patientID, clinicID, timeframe string) ([]model.Appointment, error) {
	appointments := []model.Appointment{}
	dbquery := s.DB.Model(model.Appointment{}).Where("user_id = ?", cu.ID)

	if patientID != "" {
		dbquery.Where("patient_id", patientID)
	}

	if clinicID != "" {
		dbquery.Where("clinic_id", clinicID)
	}

	switch timeframe {
	case "day":
		dbquery.Scopes(model.AppointmentBookedToday)
	case "week":
		dbquery.Scopes(model.AppointmentBookedThisWeek)
	case "month":
		dbquery.Scopes(model.AppointmentBookedThisMonth)
	default:
		dbquery.Where("booked_at > ?", common.BoD(time.Now()))
	}

	dbquery.Preload("Clinic").
		Preload("Patient").
		Order("booked_at desc").
		Find(&appointments)

	return appointments, nil
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

		s.DB.Model(&model.Appointment{}).
			Where("id = ?", appointment.ID).
			Update("BookedAlertSentAt", time.Now())

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

		s.DB.Model(&model.Appointment{}).
			Where("id = ?", appointment.ID).
			Update("ReminderAlertSentAt", time.Now())

		alert := model.Alert{
			Medium: model.AlertMediumEmail,
			Name:   "appointment_reminder",
			Status: model.AlertSent,
			To:     appointment.Patient.Email,
		}
		s.DB.Model(&appointment).Association("Alerts").Append(&alert)
	})
}
