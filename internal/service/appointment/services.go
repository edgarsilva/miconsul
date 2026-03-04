package appointment

import (
	"context"
	"strings"
	"time"

	"miconsul/internal/lib/libtime"
	"miconsul/internal/model"
	"miconsul/internal/server"

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

func (s *service) TakePatientByID(ctx context.Context, userID, patientID string) (model.Patient, error) {
	patient := model.Patient{ID: patientID, UserID: userID}
	patient, err := gorm.G[model.Patient](s.DB.GormDB()).
		Where("id = ? AND user_id = ?", patientID, userID).
		Take(ctx)
	if err != nil {
		return model.Patient{}, err
	}

	return patient, nil
}

func (s *service) TakeClinicByID(ctx context.Context, userID, clinicID string) (model.Clinic, error) {
	clinic := model.Clinic{ID: clinicID, UserID: userID}
	clinic, err := gorm.G[model.Clinic](s.DB.GormDB()).
		Where("id = ? AND user_id = ?", clinicID, userID).
		Take(ctx)
	if err != nil {
		return model.Clinic{}, err
	}

	return clinic, nil
}

func (s *service) CreateAppointment(ctx context.Context, appointment *model.Appointment) error {
	return gorm.G[model.Appointment](s.DB.GormDB()).Create(ctx, appointment)
}

func (s *service) TakeAppointmentByID(ctx context.Context, userID, appointmentID string) (model.Appointment, error) {
	appointment := model.Appointment{ID: appointmentID}
	err := s.DB.WithContext(ctx).
		Model(&model.Appointment{}).
		Preload("Clinic").
		Preload("Patient").
		Where("id = ? AND user_id = ?", appointmentID, userID).
		Take(&appointment).
		Error
	if err != nil {
		return model.Appointment{}, err
	}

	return appointment, nil
}

func (s *service) UpdateAppointmentByIDAndUserID(ctx context.Context, userID, appointmentID string, updates model.Appointment) error {
	rowsAffected, err := gorm.G[model.Appointment](s.DB.GormDB()).
		Where("id = ? AND user_id = ?", appointmentID, userID).
		Updates(ctx, updates)
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (s *service) DeleteAppointmentByIDAndUserID(ctx context.Context, userID, appointmentID string) error {
	rowsAffected, err := gorm.G[model.Appointment](s.DB.GormDB()).
		Where("id = ? AND user_id = ?", appointmentID, userID).
		Delete(ctx)
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (s *service) TakeAppointmentByIDAndToken(ctx context.Context, appointmentID, token string) (model.Appointment, error) {
	appointment := model.Appointment{}
	err := s.DB.WithContext(ctx).
		Preload("Clinic").
		Preload("User").
		Where("id = ? AND token = ?", appointmentID, token).
		Take(&appointment).
		Error
	if err != nil {
		return model.Appointment{}, err
	}

	return appointment, nil
}

func (s *service) TakePatientByIDWithLastDoneAppointment(ctx context.Context, userID, patientID string) (model.Patient, error) {
	patient := model.Patient{ID: patientID, UserID: userID}
	err := s.DB.Model(&model.Patient{}).
		Where("id = ? AND user_id = ?", patientID, userID).
		Preload("Appointments", func(tx *gorm.DB) *gorm.DB {
			return tx.Limit(1).Where("status = ?", model.ApntStatusDone).Order("booked_at desc")
		}).
		Take(&patient).Error
	if err != nil {
		return model.Patient{}, err
	}

	return patient, nil
}

func (s *service) UpdateAppointmentByIDAndToken(ctx context.Context, appointmentID, token string, selectColumns []string, updates model.Appointment) error {
	selectedFields := strings.Join(selectColumns, ",")

	rowsAffected, err := gorm.G[model.Appointment](s.DB.GormDB()).
		Select(selectedFields).
		Where("id = ? AND token = ?", appointmentID, token).
		Updates(ctx, updates)
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (s *service) ConfirmAppointmentByIDAndToken(ctx context.Context, appointmentID, token string) error {
	updates := model.Appointment{
		ConfirmedAt: time.Now(),
		Status:      model.ApntStatusConfirmed,
	}
	return s.UpdateAppointmentByIDAndToken(ctx, appointmentID, token, []string{"ConfirmedAt", "Status"}, updates)
}

func (s *service) CancelAppointmentByIDAndToken(ctx context.Context, appointmentID, token string) error {
	updates := model.Appointment{
		CanceledAt: time.Now(),
		Status:     model.ApntStatusCanceled,
	}
	return s.UpdateAppointmentByIDAndToken(ctx, appointmentID, token, []string{"CanceledAt", "Status"}, updates)
}

func (s *service) RequestAppointmentDateChangeByIDAndToken(ctx context.Context, appointmentID, token string) error {
	updates := model.Appointment{
		PendingAt: time.Now(),
		Status:    model.ApntStatusPending,
	}
	return s.UpdateAppointmentByIDAndToken(ctx, appointmentID, token, []string{"PendingAt", "Status"}, updates)
}

func (s *service) FindAppointmentsBy(ctx context.Context, userID, patientID, clinicID, timeframe string) ([]model.Appointment, error) {
	appointments := []model.Appointment{}
	dbquery := s.DB.Model(&model.Appointment{}).Where("user_id = ?", userID)

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
