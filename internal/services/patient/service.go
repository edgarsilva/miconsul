package patient

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"miconsul/internal/models"
	"miconsul/internal/server"

	"gorm.io/gorm"
)

type service struct {
	*server.Server
}

var ErrIDRequired = errors.New("id is required")

var ErrInvalidFilename = errors.New("invalid filename")

var ErrProfilePicNotProvided = errors.New("profile picture not provided")

var e164Pattern = regexp.MustCompile(`^\+[1-9]\d{7,14}$`)

func NewService(s *server.Server) (service, error) {
	if s == nil {
		return service{}, errors.New("patient service requires a non-nil server")
	}

	return service{
		Server: s,
	}, nil
}

func (s service) TakePatientByID(ctx context.Context, userID uint, patientID string) (models.Patient, error) {
	patientID = strings.TrimSpace(patientID)
	if patientID == "" {
		return models.Patient{}, ErrIDRequired
	}

	return gorm.G[models.Patient](s.DB.GormDB()).
		Where("uid = ? AND user_id = ?", patientID, userID).
		Take(ctx)
}

func (s service) FindRecentPatientsByUser(ctx context.Context, cu models.User, limit int) ([]models.Patient, error) {
	patients := []models.Patient{}
	err := s.DB.WithContext(ctx).
		Model(&cu).
		Order("created_at desc").
		Limit(limit).
		Association("Patients").
		Find(&patients)

	if err != nil {
		return []models.Patient{}, err
	}

	return patients, nil
}

func (s service) SearchPatientsByUser(ctx context.Context, cu models.User, query string, limit int) ([]models.Patient, error) {
	query = strings.TrimSpace(query)
	patients := []models.Patient{}

	dbquery := s.DB.WithContext(ctx).Model(&cu)
	if query != "" {
		dbquery.Scopes(models.GlobalFTS(query))
	} else {
		dbquery.Order("created_at desc")
	}

	err := dbquery.Limit(limit).Association("Patients").Find(&patients)
	if err != nil {
		return []models.Patient{}, err
	}

	return patients, nil
}

// FindLastAppointmentsByPatient returns the booked time of each patient's
// latest done appointment, keyed by patient ID. Patients without one are
// left out.
func (s service) FindLastAppointmentsByPatient(ctx context.Context, userID uint, patients []models.Patient) (map[uint]time.Time, error) {
	lastAppointments := map[uint]time.Time{}
	if len(patients) == 0 {
		return lastAppointments, nil
	}

	patientIDs := make([]uint, len(patients))
	for i, patient := range patients {
		patientIDs[i] = patient.ID
	}

	rows := []struct {
		PatientID uint
		BookedAt  time.Time
	}{}
	err := gorm.G[models.Appointment](s.DB.GormDB()).
		Select("patient_id, booked_at").
		Where("user_id = ? AND patient_id IN ? AND status = ?", userID, patientIDs, models.AppointmentDone).
		Where(`booked_at = (
			SELECT MAX(latest.booked_at) FROM appointments AS latest
			WHERE latest.user_id = appointments.user_id
				AND latest.patient_id = appointments.patient_id
				AND latest.status = appointments.status
				AND latest.deleted_at IS NULL
		)`).
		Scan(ctx, &rows)
	if err != nil {
		return lastAppointments, fmt.Errorf("find last done appointments: %w", err)
	}

	for _, row := range rows {
		lastAppointments[row.PatientID] = row.BookedAt
	}

	return lastAppointments, nil
}

func (s service) CreatePatient(ctx context.Context, patient *models.Patient) error {
	err := normalizePatient(patient)
	if err != nil {
		return err
	}

	return gorm.G[models.Patient](s.DB.GormDB()).Create(ctx, patient)
}

// UpdatePatientByID writes the patient form's columns, so cleared fields save as empty.
func (s service) UpdatePatientByID(ctx context.Context, userID uint, patientID string, patient models.Patient) error {
	patientID = strings.TrimSpace(patientID)
	if patientID == "" {
		return ErrIDRequired
	}
	err := normalizePatient(&patient)
	if err != nil {
		return err
	}

	rowsAffected, err := gorm.G[models.Patient](s.DB.GormDB()).
		Where("uid = ? AND user_id = ?", patientID, userID).
		Select(patientFormColumns[0], patientFormColumns[1:]).
		Updates(ctx, patient)
	if err != nil {
		return err
	}
	if rowsAffected == 1 {
		return nil
	}

	exists, err := s.patientExistsByID(ctx, userID, patientID)
	if err != nil {
		return err
	}
	if !exists {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (s service) DeletePatientByID(ctx context.Context, userID uint, patientID string) error {
	patientID = strings.TrimSpace(patientID)
	if patientID == "" {
		return ErrIDRequired
	}

	rowsAffected, err := gorm.G[models.Patient](s.DB.GormDB()).
		Where("uid = ? AND user_id = ?", patientID, userID).
		Delete(ctx)
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (s service) ClearPatientProfilePic(ctx context.Context, userID uint, patientID string) error {
	return s.UpdatePatientProfilePic(ctx, userID, patientID, "")
}

// UpdatePatientProfilePic sets the patient's profile picture path.
func (s service) UpdatePatientProfilePic(ctx context.Context, userID uint, patientID string, path string) error {
	patientID = strings.TrimSpace(patientID)
	if patientID == "" {
		return ErrIDRequired
	}

	rowsAffected, err := gorm.G[models.Patient](s.DB.GormDB()).
		Where("uid = ? AND user_id = ?", patientID, userID).
		Update(ctx, "profile_pic", path)
	if err != nil {
		return fmt.Errorf("update patient profile pic: %w", err)
	}
	if rowsAffected == 1 {
		return nil
	}

	exists, err := s.patientExistsByID(ctx, userID, patientID)
	if err != nil {
		return err
	}
	if !exists {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (s service) patientExistsByID(ctx context.Context, userID uint, patientID string) (bool, error) {
	patientID = strings.TrimSpace(patientID)
	if patientID == "" {
		return false, ErrIDRequired
	}

	var count int64
	err := s.DB.WithContext(ctx).
		Model(&models.Patient{}).
		Where("uid = ? AND user_id = ?", patientID, userID).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s service) CreatePatientsInBatches(ctx context.Context, patients []models.Patient, batchSize int) (int64, error) {
	result := s.DB.WithContext(ctx).CreateInBatches(&patients, batchSize)
	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

func normalizePatient(patient *models.Patient) error {
	if patient == nil {
		return errors.New("patient is required")
	}

	patient.Name = strings.TrimSpace(patient.Name)
	patient.Email = strings.ToLower(strings.TrimSpace(patient.Email))
	patient.Phone = normalizePatientPhone(strings.TrimSpace(patient.Phone))

	if len(patient.Name) > 120 {
		return errors.New("name exceeds max length")
	}
	if len(patient.Email) > 254 {
		return errors.New("email exceeds max length")
	}
	if len(patient.Phone) > 40 {
		return errors.New("phone exceeds max length")
	}

	return nil
}

func normalizePatientPhone(value string) string {
	original := strings.TrimSpace(value)
	if original == "" {
		return ""
	}

	normalized := strings.TrimPrefix(original, "whatsapp:")
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ReplaceAll(normalized, "(", "")
	normalized = strings.ReplaceAll(normalized, ")", "")

	if normalized == "" {
		return ""
	}

	if strings.HasPrefix(normalized, "+") {
		if e164Pattern.MatchString(normalized) {
			return normalized
		}
		return original
	}

	if len(normalized) == 10 {
		normalized = "+52" + normalized
	} else if strings.HasPrefix(normalized, "52") {
		normalized = "+" + normalized
	} else {
		normalized = "+" + normalized
	}

	if e164Pattern.MatchString(normalized) {
		return normalized
	}

	return original
}
