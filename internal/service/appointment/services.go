package appointment

import (
	"errors"
	"fmt"
	"miconsul/internal/lib/libtime"
	"miconsul/internal/mailer"
	"miconsul/internal/model"
	"miconsul/internal/server"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
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
	result := s.DB.Where(&patient, "ID", "UserID").Take(&patient)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
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
	result := s.DB.Where(&clinic, "ID", "UserID").Take(&clinic)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return model.Clinic{}, errors.New("incorrect number of clinic rows, expecting: 1, got:" + string(result.RowsAffected))
	}

	return clinic, nil
}

func (s *service) GetAppointmentsBy(c *fiber.Ctx, cu model.User, patientID, clinicID, timeframe string) ([]model.Appointment, error) {
	appointments := []model.Appointment{}
	dbquery := s.DB.Model(model.Appointment{}).Where("user_id = ?", cu.ID)

	if patientID != "" {
		dbquery.Where("patient_id = ?", patientID)
	}

	if clinicID != "" {
		dbquery.Where("clinic_id = ?", clinicID)
	}

	switch timeframe {
	case "day":
		dbquery.Scopes(model.AppointmentBookedToday)
	case "week":
		dbquery.Scopes(model.AppointmentBookedThisWeek)
	case "month":
		dbquery.Scopes(model.AppointmentBookedThisMonth)
	default:
		dbquery.Where("booked_at > ?", libtime.BoD(time.Now()))
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
