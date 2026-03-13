package clinic

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

func NewService(s *server.Server) (service, error) {
	if s == nil {
		return service{}, errors.New("clinic service requires a non-nil server")
	}

	return service{
		Server: s,
	}, nil
}

func (s service) TakeClinicByID(ctx context.Context, userID, clinicID string) (model.Clinic, error) {
	clinicID = strings.TrimSpace(clinicID)
	if clinicID == "" {
		return model.Clinic{}, ErrIDRequired
	}

	clinic, err := gorm.G[model.Clinic](s.DB.GormDB()).
		Where("id = ? AND user_id = ?", clinicID, userID).
		Take(ctx)
	if err != nil {
		return model.Clinic{}, err
	}

	return clinic, nil
}

func (s service) FindClinicsBySearchTerm(ctx context.Context, cu model.User, searchTerm string) ([]model.Clinic, error) {
	clinics := []model.Clinic{}
	searchTerm = strings.TrimSpace(searchTerm)

	err := s.DB.WithContext(ctx).
		Model(&cu).
		Scopes(model.GlobalFTS(searchTerm)).
		Limit(QUERY_LIMIT).
		Association("Clinics").
		Find(&clinics)
	if err != nil {
		return []model.Clinic{}, err
	}

	return clinics, nil
}

func (s service) CreateClinic(ctx context.Context, clinic *model.Clinic) error {
	err := normalizeClinicWriteInput(clinic)
	if err != nil {
		return err
	}

	return gorm.G[model.Clinic](s.DB.GormDB()).Create(ctx, clinic)
}

func (s service) UpdateClinicByID(ctx context.Context, userID, clinicID string, clinic model.Clinic) error {
	clinicID = strings.TrimSpace(clinicID)
	if clinicID == "" {
		return ErrIDRequired
	}

	err := normalizeClinicWriteInput(&clinic)
	if err != nil {
		return err
	}

	rowsAffected, err := gorm.G[model.Clinic](s.DB.GormDB()).
		Where("id = ? AND user_id = ?", clinicID, userID).
		Updates(ctx, clinic)
	if err != nil {
		return err
	}
	if rowsAffected == 1 {
		return nil
	}

	exists, err := s.clinicExistsByID(ctx, userID, clinicID)
	if err != nil {
		return err
	}
	if !exists {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (s service) DeleteClinicByID(ctx context.Context, userID, clinicID string) error {
	clinicID = strings.TrimSpace(clinicID)
	if clinicID == "" {
		return ErrIDRequired
	}

	rowsAffected, err := gorm.G[model.Clinic](s.DB.GormDB()).
		Where("id = ? AND user_id = ?", clinicID, userID).
		Delete(ctx)
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (s service) clinicExistsByID(ctx context.Context, userID, clinicID string) (bool, error) {
	clinicID = strings.TrimSpace(clinicID)
	if clinicID == "" {
		return false, ErrIDRequired
	}

	var count int64
	err := s.DB.WithContext(ctx).
		Model(&model.Clinic{}).
		Where("id = ? AND user_id = ?", clinicID, userID).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func normalizeClinicWriteInput(clinic *model.Clinic) error {
	if clinic == nil {
		return errors.New("clinic is required")
	}

	clinic.Name = strings.TrimSpace(clinic.Name)
	clinic.Email = strings.ToLower(strings.TrimSpace(clinic.Email))
	clinic.Phone = strings.TrimSpace(clinic.Phone)

	if len(clinic.Name) > 120 {
		return errors.New("name exceeds max length")
	}
	if len(clinic.Email) > 254 {
		return errors.New("email exceeds max length")
	}
	if len(clinic.Phone) > 40 {
		return errors.New("phone exceeds max length")
	}

	return nil
}
