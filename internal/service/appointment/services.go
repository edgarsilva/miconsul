package appointment

import (
	"context"
	"fmt"
	"miconsul/internal/lib/libtime"
	"miconsul/internal/mailer"
	"miconsul/internal/model"
	"miconsul/internal/server"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type service struct {
	*server.Server
}

func NewService(s *server.Server) service {
	ser := service{Server: s}
	ser.RegisterCronJob()

	return ser
}

func (s *service) RegisterCronJob() {
	err := s.AddCronJob("0/1 * * * *", func() {
		ctx, span := s.Trace(context.Background(), "appointment/services:RegisterCronJob>Job",
			trace.WithAttributes(
				attribute.String("grouping.fingerprint", "cronjob"),
			),
		)
		defer span.End()

		appointments := []model.Appointment{}
		if err := s.DB.
			WithContext(ctx).
			Model(&model.Appointment{}).
			Preload("Patient").
			Preload("Clinic").
			Scopes(model.AppointmentWithPendingAlerts).
			Find(&appointments).
			Error; err != nil {
			fmt.Println("failed to load appointments for reminder job:", err.Error())
			return
		}
		for _, appointment := range appointments {
			s.SendReminderAlert(appointment)
		}
	})
	if err != nil {
		fmt.Println(err.Error())
	}
}

func (s *service) GetPatientByID(c fiber.Ctx, id string) (model.Patient, error) {
	if id == "" {
		return model.Patient{}, nil
	}

	cu, err := s.CurrentUser(c)
	if err != nil {
		return model.Patient{}, err
	}

	patient := model.Patient{ID: id, UserID: cu.ID}
	patient, err = gorm.G[model.Patient](s.DB.GormDB()).
		Where("id = ? AND user_id = ?", id, cu.ID).
		Take(c.Context())
	if err != nil {
		return model.Patient{}, err
	}

	return patient, nil
}

func (s *service) GetClinicByID(c fiber.Ctx, id string) (model.Clinic, error) {
	if id == "" {
		return model.Clinic{}, nil
	}

	cu, err := s.CurrentUser(c)
	if err != nil {
		return model.Clinic{}, err
	}

	clinic := model.Clinic{ID: id, UserID: cu.ID}
	clinic, err = gorm.G[model.Clinic](s.DB.GormDB()).
		Where("id = ? AND user_id = ?", id, cu.ID).
		Take(c.Context())
	if err != nil {
		return model.Clinic{}, err
	}

	return clinic, nil
}

func (s *service) TakeAppointmentByID(ctx context.Context, userID, appointmentID string) (model.Appointment, error) {
	appointment, err := gorm.G[model.Appointment](s.DB.GormDB()).
		Where("id = ? AND user_id = ?", appointmentID, userID).
		Take(ctx)
	if err != nil {
		return model.Appointment{}, err
	}

	return appointment, nil
}

func (s *service) AppointmentForShowPage(ctx context.Context, userID, appointmentID string) (model.Appointment, error) {
	appointment := model.Appointment{ID: appointmentID}
	if appointmentID == "" || appointmentID == "new" {
		return appointment, nil
	}

	return s.TakeAppointmentByID(ctx, userID, appointmentID)
}

func (s *service) TakePatientByIDWithLastDoneAppointment(ctx context.Context, userID, patientID string) (model.Patient, error) {
	patient := model.Patient{ID: patientID, UserID: userID}
	if err := s.DB.Model(&model.Patient{}).
		Where("id = ? AND user_id = ?", patientID, userID).
		Preload("Appointments", func(tx *gorm.DB) *gorm.DB {
			return tx.Limit(1).Where("status = ?", model.ApntStatusDone).Order("booked_at desc")
		}).
		Take(&patient).Error; err != nil {
		return model.Patient{}, err
	}

	return patient, nil
}

func (s *service) PatientForStartPage(ctx context.Context, userID, patientID string) (model.Patient, error) {
	if patientID == "" {
		return model.Patient{}, gorm.ErrRecordNotFound
	}

	return s.TakePatientByIDWithLastDoneAppointment(ctx, userID, patientID)
}

func (s *service) GetAppointmentsBy(c fiber.Ctx, cu model.User, patientID, clinicID, timeframe string) ([]model.Appointment, error) {
	appointments := []model.Appointment{}
	dbquery := s.DB.Model(&model.Appointment{}).Where("user_id = ?", cu.ID)

	if patientID != "" {
		dbquery = dbquery.Where("patient_id = ?", patientID)
	}

	if clinicID != "" {
		dbquery = dbquery.Where("clinic_id = ?", clinicID)
	}

	switch timeframe {
	case "day":
		dbquery = dbquery.Scopes(model.AppointmentBookedToday)
	case "week":
		dbquery = dbquery.Scopes(model.AppointmentBookedThisWeek)
	case "month":
		dbquery = dbquery.Scopes(model.AppointmentBookedThisMonth)
	default:
		dbquery = dbquery.Where("booked_at > ?", libtime.BoD(time.Now()))
	}

	err := dbquery.Preload("Clinic").
		Preload("Patient").
		Order("booked_at desc").
		Find(&appointments).
		Error
	if err != nil {
		return nil, err
	}

	return appointments, nil
}

func (s *service) SendBookedAlert(appointment model.Appointment) error {
	err := s.SendToWorker(func() {
		err := mailer.SendAppointmentBookedEmail(appointment)
		if err != nil {
			alert := model.Alert{
				Medium: model.AlertMediumEmail,
				Name:   "appointment_booked",
				Status: model.AlertFailed,
				To:     appointment.Patient.Email,
			}
			if err := s.DB.Model(&appointment).Association("Alerts").Append(&alert); err != nil {
				fmt.Println("failed to append failed booked alert:", err.Error())
			}
			return
		}

		if _, err := gorm.G[model.Appointment](s.DB.GormDB()).
			Where("id = ?", appointment.ID).
			Update(context.Background(), "BookedAlertSentAt", time.Now()); err != nil {
			fmt.Println("failed to update BookedAlertSentAt:", err.Error())
		}

		alert := model.Alert{
			Medium: model.AlertMediumEmail,
			Name:   "appointment_booked",
			Status: model.AlertSent,
			To:     appointment.Patient.Email,
		}
		if err := s.DB.Model(&appointment).Association("Alerts").Append(&alert); err != nil {
			fmt.Println("failed to append sent booked alert:", err.Error())
		}
	})

	return err
}

func (s *service) SendReminderAlert(appointment model.Appointment) error {
	err := s.SendToWorker(func() {
		err := mailer.SendAppointmentReminderEmail(appointment)
		if err != nil {
			alert := model.Alert{
				Medium: model.AlertMediumEmail,
				Name:   "appointment_reminder",
				Status: model.AlertFailed,
				To:     appointment.Patient.Email,
			}
			if err := s.DB.Model(&appointment).Association("Alerts").Append(&alert); err != nil {
				fmt.Println("failed to append failed reminder alert:", err.Error())
			}
			return
		}

		if _, err := gorm.G[model.Appointment](s.DB.GormDB()).
			Where("id = ?", appointment.ID).
			Update(context.Background(), "ReminderAlertSentAt", time.Now()); err != nil {
			fmt.Println("failed to update ReminderAlertSentAt:", err.Error())
		}

		alert := model.Alert{
			Medium: model.AlertMediumEmail,
			Name:   "appointment_reminder",
			Status: model.AlertSent,
			To:     appointment.Patient.Email,
		}
		if err := s.DB.Model(&appointment).Association("Alerts").Append(&alert); err != nil {
			fmt.Println("failed to append sent reminder alert:", err.Error())
		}
	})

	return err
}
