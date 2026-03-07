package patient

import (
	"context"
	"errors"
	"miconsul/internal/model"
	"miconsul/internal/server"
	"strings"

	"gorm.io/gorm"
)

type service struct {
	*server.Server
}

var ErrIDRequired = errors.New("id is required")

var ErrInvalidFilename = errors.New("invalid filename")

var ErrProfilePicNotProvided = errors.New("profile picture not provided")

func NewService(s *server.Server) (service, error) {
	if s == nil {
		return service{}, errors.New("patient service requires a non-nil server")
	}

	return service{
		Server: s,
	}, nil
}

func (s service) TakePatientByID(ctx context.Context, userID, patientID string) (model.Patient, error) {
	patientID = strings.TrimSpace(patientID)
	if patientID == "" {
		return model.Patient{}, ErrIDRequired
	}

	return gorm.G[model.Patient](s.DB.GormDB()).
		Where("id = ? AND user_id = ?", patientID, userID).
		Take(ctx)
}

func (s service) PatientForShowPage(ctx context.Context, userID, patientID string) (model.Patient, error) {
	patientID = strings.TrimSpace(patientID)
	patient := model.Patient{}
	if patientID == "" || patientID == "new" {
		return patient, nil
	}

	return s.TakePatientByID(ctx, userID, patientID)
}

func (s service) FindRecentPatientsByUser(ctx context.Context, cu model.User, limit int) ([]model.Patient, error) {
	patients := []model.Patient{}
	err := s.DB.WithContext(ctx).
		Model(&cu).
		Order("created_at desc").
		Limit(limit).
		Association("Patients").
		Find(&patients)

	if err != nil {
		return []model.Patient{}, err
	}

	return patients, nil
}

func (s service) SearchPatientsByUser(ctx context.Context, cu model.User, query string, limit int) ([]model.Patient, error) {
	query = strings.TrimSpace(query)
	patients := []model.Patient{}

	dbquery := s.DB.WithContext(ctx).Model(&cu)
	if query != "" {
		dbquery.Scopes(model.GlobalFTS(query))
	} else {
		dbquery.Order("created_at desc")
	}

	err := dbquery.Limit(limit).Association("Patients").Find(&patients)
	if err != nil {
		return []model.Patient{}, err
	}

	return patients, nil
}

func (s service) CreatePatient(ctx context.Context, patient *model.Patient) error {
	return gorm.G[model.Patient](s.DB.GormDB()).Create(ctx, patient)
}

func (s service) UpdatePatientByID(ctx context.Context, userID, patientID string, patient model.Patient) error {
	patientID = strings.TrimSpace(patientID)
	if patientID == "" {
		return ErrIDRequired
	}

	rowsAffected, err := gorm.G[model.Patient](s.DB.GormDB()).
		Where("id = ? AND user_id = ?", patientID, userID).
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

func (s service) DeletePatientByID(ctx context.Context, userID, patientID string) error {
	patientID = strings.TrimSpace(patientID)
	if patientID == "" {
		return ErrIDRequired
	}

	rowsAffected, err := gorm.G[model.Patient](s.DB.GormDB()).
		Where("id = ? AND user_id = ?", patientID, userID).
		Delete(ctx)
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (s service) ClearPatientProfilePic(ctx context.Context, userID, patientID string) error {
	patientID = strings.TrimSpace(patientID)
	if patientID == "" {
		return ErrIDRequired
	}

	rowsAffected, err := gorm.G[model.Patient](s.DB.GormDB()).
		Where("id = ? AND user_id = ?", patientID, userID).
		Update(ctx, "profile_pic", "")
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

func (s service) patientExistsByID(ctx context.Context, userID, patientID string) (bool, error) {
	patientID = strings.TrimSpace(patientID)
	if patientID == "" {
		return false, ErrIDRequired
	}

	var count int64
	err := s.DB.WithContext(ctx).
		Model(&model.Patient{}).
		Where("id = ? AND user_id = ?", patientID, userID).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s service) CreatePatientsInBatches(ctx context.Context, patients []model.Patient, batchSize int) (int64, error) {
	result := s.DB.WithContext(ctx).CreateInBatches(&patients, batchSize)
	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}
