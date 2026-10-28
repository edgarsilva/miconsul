// Package appointment provides the appointments business logic.
package appointment

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"miconsul/internal/models"
	"miconsul/internal/server"

	"gorm.io/gorm"
)

type service struct {
	*server.Server
}

var (
	ErrIDRequired    = errors.New("id is required")
	ErrTokenRequired = errors.New("token is required")
	// ErrAppointmentChanged means the appointment's status changed after it
	// was loaded, so the action was not applied.
	ErrAppointmentChanged = errors.New("appointment changed since it was loaded")
)

const (
	defaultWorkerContextTimeout  = 10 * time.Second
	defaultCronJobContextTimeout = 45 * time.Second
)

func New(s *server.Server) (*service, error) {
	if s == nil {
		return nil, errors.New("appointment service requires a non-nil server")
	}

	svc := &service{Server: s}
	if err := svc.bootstrapJobs(); err != nil {
		return nil, err
	}

	return svc, nil
}

func (s *service) TakePatientByUID(ctx context.Context, userID uint, patientUID string) (models.Patient, error) {
	if strings.TrimSpace(patientUID) == "" {
		return models.Patient{}, ErrIDRequired
	}

	patient, err := gorm.G[models.Patient](s.DB.GormDB()).
		Where("uid = ? AND user_id = ?", patientUID, userID).
		Take(ctx)
	if err != nil {
		return models.Patient{}, err
	}

	return patient, nil
}

func (s *service) TakeClinicByUID(ctx context.Context, userID uint, clinicUID string) (models.Clinic, error) {
	if strings.TrimSpace(clinicUID) == "" {
		return models.Clinic{}, ErrIDRequired
	}

	clinic, err := gorm.G[models.Clinic](s.DB.GormDB()).
		Where("uid = ? AND user_id = ?", clinicUID, userID).
		Take(ctx)
	if err != nil {
		return models.Clinic{}, err
	}

	return clinic, nil
}

func (s *service) CreateAppointment(ctx context.Context, appointment *models.Appointment) error {
	if appointment == nil {
		return errors.New("appointment is required")
	}

	if appointment.Status == "" {
		appointment.Status = models.AppointmentPending
	}

	if !appointment.Status.IsValid() {
		return errors.New("invalid appointment status")
	}

	if err := gorm.G[models.Appointment](s.DB.GormDB()).Create(ctx, appointment); err != nil {
		return err
	}

	if appointment.UID != "" {
		_ = s.recordFeedEvent(ctx, appointment.UserID, appointment.UID, models.EventActionCreated)
	}

	return nil
}

func (s *service) TakeAppointmentByUID(ctx context.Context, userID uint, appointmentUID string) (models.Appointment, error) {
	if strings.TrimSpace(appointmentUID) == "" {
		return models.Appointment{}, ErrIDRequired
	}

	appointment, err := gorm.G[models.Appointment](s.DB.GormDB()).
		Preload("Clinic", nil).
		Preload("Patient", nil).
		Where("uid = ? AND user_id = ?", appointmentUID, userID).
		Take(ctx)
	if err != nil {
		return models.Appointment{}, err
	}

	return appointment, nil
}

func (s *service) UpdateAppointmentByID(ctx context.Context, userID uint, appointmentID string, updates appointmentPatchUpdates) error {
	if strings.TrimSpace(appointmentID) == "" {
		return ErrIDRequired
	}

	columns := []string{
		"BookedAt",
		"BookedYear",
		"BookedMonth",
		"BookedDay",
		"BookedHour",
		"BookedMinute",
		"Price",
		"ClinicID",
		"PatientID",
		"Duration",
	}

	if err := s.updateAppointmentColumnsByID(ctx, userID, appointmentID, columns, &updates); err != nil {
		return err
	}

	_ = s.recordFeedEvent(ctx, userID, appointmentID, models.EventActionUpdated)
	return nil
}

func (s *service) CompleteAppointmentByID(ctx context.Context, userID uint, appointmentID string, updates appointmentCompleteUpdates) error {
	if strings.TrimSpace(appointmentID) == "" {
		return ErrIDRequired
	}

	if !updates.Status.IsValid() {
		return errors.New("invalid appointment status")
	}

	columns := []string{"Status", "Observations", "Conclusions", "Summary", "Notes"}

	if err := s.updateAppointmentColumnsByID(ctx, userID, appointmentID, columns, &updates); err != nil {
		return err
	}

	_ = s.recordFeedEvent(ctx, userID, appointmentID, models.EventActionCompleted)
	return nil
}

func (s *service) SaveAppointmentSessionByID(ctx context.Context, userID uint, appointmentID string, updates appointmentSessionUpdates) error {
	if strings.TrimSpace(appointmentID) == "" {
		return ErrIDRequired
	}

	columns := []string{"Observations", "Conclusions", "Summary", "Notes"}

	if err := s.updateAppointmentColumnsByID(ctx, userID, appointmentID, columns, &updates); err != nil {
		return err
	}

	_ = s.recordFeedEvent(ctx, userID, appointmentID, models.EventActionUpdated)
	return nil
}

func (s *service) StartAppointmentSessionByID(ctx context.Context, userID uint, appointmentID string) error {
	if strings.TrimSpace(appointmentID) == "" {
		return ErrIDRequired
	}

	updates := appointmentCompleteUpdates{Status: models.AppointmentInProgress}
	if err := s.updateAppointmentColumnsByID(ctx, userID, appointmentID, []string{"Status"}, &updates); err != nil {
		return err
	}

	_ = s.recordFeedEvent(ctx, userID, appointmentID, models.EventActionUpdated)
	return nil
}

func (s *service) PauseAppointmentSessionByID(ctx context.Context, userID uint, appointmentID string) error {
	if strings.TrimSpace(appointmentID) == "" {
		return ErrIDRequired
	}

	updates := appointmentPauseUpdates{Status: models.AppointmentPending, PendingAt: time.Now().UTC()}
	if err := s.updateAppointmentColumnsByID(ctx, userID, appointmentID, []string{"Status", "PendingAt"}, &updates); err != nil {
		return err
	}

	_ = s.recordFeedEvent(ctx, userID, appointmentID, models.EventActionUpdated)
	return nil
}

func (s *service) CancelAppointmentByID(ctx context.Context, userID uint, appointmentID string, updates appointmentCancelUpdates) error {
	if strings.TrimSpace(appointmentID) == "" {
		return ErrIDRequired
	}

	if !updates.Status.IsValid() {
		return errors.New("invalid appointment status")
	}

	columns := []string{"Status", "CanceledAt"}

	if err := s.updateAppointmentColumnsByID(ctx, userID, appointmentID, columns, &updates); err != nil {
		return err
	}

	_ = s.recordFeedEvent(ctx, userID, appointmentID, models.EventActionCanceled)
	return nil
}

func (s *service) updateAppointmentColumnsByID(ctx context.Context, userID uint, appointmentID string, columns []string, updates any) error {
	if len(columns) == 0 {
		return errors.New("columns are required")
	}

	result := s.DB.WithContext(ctx).
		Model(&models.Appointment{}).
		Where("uid = ? AND user_id = ?", appointmentID, userID).
		Select(columns).
		Omit("UserID", "Clinic", "Patient", "User", "FeedEvents", "Notifications").
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (s *service) DeleteAppointmentByID(ctx context.Context, userID uint, appointmentID string) error {
	if strings.TrimSpace(appointmentID) == "" {
		return ErrIDRequired
	}

	_ = s.recordFeedEvent(ctx, userID, appointmentID, models.EventActionDeleted)

	rowsAffected, err := gorm.G[models.Appointment](s.DB.GormDB()).
		Where("uid = ? AND user_id = ?", appointmentID, userID).
		Delete(ctx)
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (s *service) TakeAppointmentByToken(ctx context.Context, token string) (models.Appointment, error) {
	if strings.TrimSpace(token) == "" {
		return models.Appointment{}, ErrTokenRequired
	}

	appointment, err := gorm.G[models.Appointment](s.DB.GormDB()).
		Preload("Clinic", nil).
		Preload("User", nil).
		Preload("Patient", nil).
		Where("token = ? AND booked_at > ?", token, patientLinkValidAfter()).
		Take(ctx)
	if err != nil {
		return models.Appointment{}, err
	}

	return appointment, nil
}

func (s *service) TakePatientByUIDWithLastDoneAppointment(ctx context.Context, userID uint, patientUID string) (models.Patient, error) {
	if strings.TrimSpace(patientUID) == "" {
		return models.Patient{}, ErrIDRequired
	}

	patient := models.Patient{}
	err := s.DB.Model(&models.Patient{}).
		Where("uid = ? AND user_id = ?", patientUID, userID).
		Preload("Appointments", func(tx *gorm.DB) *gorm.DB {
			return tx.Limit(1).Where("status = ?", models.AppointmentDone).Order("booked_at desc")
		}).
		Take(&patient).Error
	if err != nil {
		return models.Patient{}, err
	}

	return patient, nil
}

// ConfirmAppointment marks a loaded appointment as confirmed by its patient
// and records a feed event. Confirming an already confirmed appointment is a
// no-op. It returns ErrAppointmentChanged when the status changed since the
// appointment was loaded.
func (s *service) ConfirmAppointment(ctx context.Context, appointment models.Appointment) (models.Appointment, error) {
	if appointment.Status == models.AppointmentConfirmed {
		return appointment, nil
	}

	now := time.Now().UTC()
	if err := s.setAppointmentStatus(ctx, appointment, models.AppointmentConfirmed, "confirmed_at", now); err != nil {
		return appointment, err
	}
	appointment.Status = models.AppointmentConfirmed
	appointment.ConfirmedAt = now

	return appointment, s.recordPatientFeedEvent(ctx, appointment, models.EventActionConfirmed)
}

// CancelAppointment marks a loaded appointment as canceled by its patient and
// records a feed event. Canceling an already canceled appointment is a no-op.
// It returns ErrAppointmentChanged when the status changed since the
// appointment was loaded.
func (s *service) CancelAppointment(ctx context.Context, appointment models.Appointment) (models.Appointment, error) {
	if appointment.Status == models.AppointmentCanceled {
		return appointment, nil
	}

	now := time.Now().UTC()
	if err := s.setAppointmentStatus(ctx, appointment, models.AppointmentCanceled, "canceled_at", now); err != nil {
		return appointment, err
	}
	appointment.Status = models.AppointmentCanceled
	appointment.CanceledAt = now

	return appointment, s.recordPatientFeedEvent(ctx, appointment, models.EventActionCanceled)
}

// RequestAppointmentDateChange marks a loaded appointment as pending a new
// date. It returns ErrAppointmentChanged when the status changed since the
// appointment was loaded.
func (s *service) RequestAppointmentDateChange(ctx context.Context, appointment models.Appointment) (models.Appointment, error) {
	now := time.Now().UTC()
	if err := s.setAppointmentStatus(ctx, appointment, models.AppointmentPending, "pending_at", now); err != nil {
		return appointment, err
	}
	appointment.Status = models.AppointmentPending
	appointment.PendingAt = now

	return appointment, nil
}

func (s *service) recordFeedEvent(ctx context.Context, userID uint, appointmentUID string, action models.EventAction) error {
	if appointmentUID == "" {
		return nil
	}

	appointment, err := s.TakeAppointmentByUID(ctx, userID, appointmentUID)
	if err != nil {
		return err
	}

	var actorName, actorID string
	if userID > 0 {
		user, err := gorm.G[models.User](s.DB.GormDB()).Where("id = ?", userID).Take(ctx)
		if err == nil {
			actorName = user.Name
			actorID = user.UID
		}
	}
	if actorName == "" {
		actorName = "Miconsul"
	}

	event := models.NewFeedEvent(action, actorName, actorID, "/users/"+actorID, appointment)
	return s.DB.WithContext(ctx).Create(&event).Error
}

func (s *service) FindAppointmentsBy(ctx context.Context, userID uint, patientID, clinicID, timeframe, searchTerm, statusFilter string) ([]models.Appointment, error) {
	appointments := []models.Appointment{}
	dbquery := s.DB.Model(&models.Appointment{}).Where("appointments.user_id = ?", userID)

	if patientID != "" {
		dbquery = dbquery.Joins("INNER JOIN patients ON patients.id = appointments.patient_id")
	}
	if patientID != "" {
		dbquery = dbquery.Where("patients.uid = ?", patientID)
	}

	if clinicID != "" {
		dbquery = dbquery.Joins("INNER JOIN clinics ON clinics.id = appointments.clinic_id")
	}
	if clinicID != "" {
		dbquery = dbquery.Where("clinics.uid = ?", clinicID)
	}

	searchTerm = strings.TrimSpace(searchTerm)
	applySearch := searchTerm != ""
	if applySearch {
		patientIDs, err := s.findPatientIDsBySearchTerm(ctx, userID, searchTerm)
		if err != nil {
			return nil, err
		}

		clinicIDs, err := s.findClinicIDsBySearchTerm(ctx, userID, searchTerm)
		if err != nil {
			return nil, err
		}

		if len(patientIDs) == 0 && len(clinicIDs) == 0 {
			return []models.Appointment{}, nil
		}

		dbquery = dbquery.Where("appointments.patient_id IN ? OR appointments.clinic_id IN ?", patientIDs, clinicIDs)
	}

	switch timeframe {
	case "day":
		dbquery = dbquery.Scopes(models.AppointmentBookedToday)
	case "week":
		dbquery = dbquery.Scopes(models.AppointmentBookedThisWeek)
	case "month":
		dbquery = dbquery.Scopes(models.AppointmentBookedThisMonth)
	default:
		dbquery = dbquery.Scopes(models.AppointmentBookedToday)
	}

	statusFilter = strings.TrimSpace(statusFilter)
	if statusFilter != "" {
		statuses := strings.Split(statusFilter, ",")
		for i := range statuses {
			statuses[i] = strings.TrimSpace(statuses[i])
		}
		dbquery = dbquery.Where("status IN ?", statuses)
	}

	err := dbquery.Preload("Clinic").
		Preload("Patient").
		Order("booked_at desc").
		Limit(20).
		Find(&appointments).
		Error
	if err != nil {
		return nil, err
	}

	return appointments, nil
}

func (s *service) findPatientIDsBySearchTerm(ctx context.Context, userID uint, searchTerm string) ([]uint, error) {
	if searchTerm == "" {
		return []uint{}, nil
	}

	var ids []uint
	err := s.DB.WithContext(ctx).
		Model(&models.Patient{}).
		Where("user_id = ?", userID).
		Scopes(models.GlobalFTS(searchTerm)).
		Limit(50).
		Pluck("patients.id", &ids).
		Error
	if err != nil {
		return nil, err
	}

	return ids, nil
}

func (s *service) findClinicIDsBySearchTerm(ctx context.Context, userID uint, searchTerm string) ([]uint, error) {
	if searchTerm == "" {
		return []uint{}, nil
	}

	var ids []uint
	err := s.DB.WithContext(ctx).
		Model(&models.Clinic{}).
		Where("user_id = ?", userID).
		Scopes(models.GlobalFTS(searchTerm)).
		Limit(50).
		Pluck("clinics.id", &ids).
		Error
	if err != nil {
		return nil, err
	}

	return ids, nil
}

func (s *service) FindClinicsBySearchTerm(ctx context.Context, userID uint, searchTerm string) ([]models.Clinic, error) {
	clinics := []models.Clinic{}
	searchTerm = strings.TrimSpace(searchTerm)
	err := s.DB.WithContext(ctx).
		Model(&models.Clinic{}).
		Where("user_id = ?", userID).
		Scopes(models.GlobalFTS(searchTerm)).
		Limit(10).
		Find(&clinics).
		Error
	if err != nil {
		return []models.Clinic{}, err
	}

	return clinics, nil
}

func (s *service) FindRecentClinicsByUserID(ctx context.Context, userID uint, limit int) ([]models.Clinic, error) {
	clinics := []models.Clinic{}
	err := s.DB.WithContext(ctx).
		Model(&models.Clinic{}).
		Where("user_id = ?", userID).
		Order("created_at desc").
		Limit(limit).
		Find(&clinics).
		Error
	if err != nil {
		return nil, err
	}

	return clinics, nil
}

func (s *service) FindRecentPatientsByUserID(ctx context.Context, userID uint, limit int) ([]models.Patient, error) {
	patients := []models.Patient{}
	err := s.DB.WithContext(ctx).
		Model(&models.Patient{}).
		Where("user_id = ?", userID).
		Order("created_at desc").
		Limit(limit).
		Find(&patients).
		Error
	if err != nil {
		return nil, err
	}

	return patients, nil
}

// patientLinkValidAfter is the earliest appointment time a patient link still
// works for.
func patientLinkValidAfter() time.Time {
	return time.Now().UTC().Add(-models.PatientLinkGrace)
}

// setAppointmentStatus sets status and stamps column with at, only while the
// appointment still has the status it was loaded with.
func (s *service) setAppointmentStatus(ctx context.Context, appointment models.Appointment, status models.AppointmentStatus, column string, at time.Time) error {
	result := s.DB.WithContext(ctx).
		Model(&models.Appointment{}).
		Where("id = ? AND status = ?", appointment.ID, appointment.Status).
		Updates(map[string]any{"status": status, column: at})
	if result.Error != nil {
		return fmt.Errorf("set appointment status: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return ErrAppointmentChanged
	}

	return nil
}

// recordPatientFeedEvent records an action the patient took on the
// appointment.
func (s *service) recordPatientFeedEvent(ctx context.Context, appointment models.Appointment, action models.EventAction) error {
	actorName := appointment.Patient.Name
	if actorName == "" {
		actorName = "Miconsul"
	}
	event := models.NewFeedEvent(action, actorName, appointment.Patient.UID, "/patients/"+appointment.Patient.UID, appointment)
	if err := s.DB.WithContext(ctx).Create(&event).Error; err != nil {
		return fmt.Errorf("record feed event: %w", err)
	}

	return nil
}

// ensureClinicListed returns clinics with the appointment's clinic added at
// the top when it isn't listed, so the edit page can show it as selected.
func ensureClinicListed(clinics []models.Clinic, appointment models.Appointment) []models.Clinic {
	listed := slices.ContainsFunc(clinics, func(clinic models.Clinic) bool { return clinic.ID == appointment.Clinic.ID })
	if appointment.Clinic.ID == 0 || listed {
		return clinics
	}

	return append([]models.Clinic{appointment.Clinic}, clinics...)
}

// ensurePatientListed returns patients with the appointment's patient added
// at the top when it isn't listed, so the edit page can show it as selected.
func ensurePatientListed(patients []models.Patient, appointment models.Appointment) []models.Patient {
	listed := slices.ContainsFunc(patients, func(patient models.Patient) bool { return patient.ID == appointment.Patient.ID })
	if appointment.Patient.ID == 0 || listed {
		return patients
	}

	return append([]models.Patient{appointment.Patient}, patients...)
}
