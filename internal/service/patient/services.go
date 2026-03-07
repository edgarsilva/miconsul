package patient

import (
	"context"
	"errors"
	"fmt"
	"miconsul/internal/model"
	"miconsul/internal/server"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type service struct {
	*server.Server
}

func NewService(s *server.Server) (service, error) {
	if s == nil {
		return service{}, errors.New("patient service requires a non-nil server")
	}

	return service{
		Server: s,
	}, nil
}

const patientsDir = "/patients"

func (s service) Patients(cu model.User, term string) ([]model.Patient, error) {
	patients := []model.Patient{}
	err := s.DB.
		Model(&cu).
		Scopes(model.GlobalFTS(term)).
		Limit(QUERY_LIMIT).
		Association("Patients").
		Find(&patients)

	return patients, err
}

func (s service) TakePatientByIDAndUserID(ctx context.Context, userID, patientID string) (model.Patient, error) {
	return gorm.G[model.Patient](s.DB.GormDB()).
		Where("id = ? AND user_id = ?", patientID, userID).
		Take(ctx)
}

func (s service) PatientForShowPage(ctx context.Context, userID, patientID string) (model.Patient, error) {
	patient := model.Patient{}
	if patientID == "" || patientID == "new" {
		return patient, nil
	}

	return s.TakePatientByIDAndUserID(ctx, userID, patientID)
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

func (s service) UpdatePatientByIDAndUserID(ctx context.Context, userID, patientID string, patient model.Patient) error {
	rowsAffected, err := gorm.G[model.Patient](s.DB.GormDB()).
		Where("id = ? AND user_id = ?", patientID, userID).
		Updates(ctx, patient)
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (s service) DeletePatientByIDAndUserID(ctx context.Context, userID, patientID string) error {
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
	rowsAffected, err := gorm.G[model.Patient](s.DB.GormDB()).
		Where("id = ? AND user_id = ?", patientID, userID).
		Update(ctx, "profile_pic", "")
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (s service) CreatePatientsInBatches(ctx context.Context, patients []model.Patient, batchSize int) (int64, error) {
	result := s.DB.WithContext(ctx).CreateInBatches(&patients, batchSize)
	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

func SaveProfilePicToDisk(c fiber.Ctx, patient model.Patient) (string, error) {
	profilePic, err := c.FormFile("profilePic")
	if err != nil {
		return "", fmt.Errorf("failed to grab profilePic from form: %w", err)
	}

	if patient.ID == "" {
		return "", errors.New("failed to save profile pic without patient.ID")
	}

	filename := patient.ID + "_ppic_" + profilePic.Filename
	path, err := ProfilePicPath(filename)
	if err != nil {
		return "", fmt.Errorf("failed to save profile pic without an ASSETS_DIR: %w", err)
	}

	err = c.SaveFile(profilePic, path)
	if err != nil {
		return "", fmt.Errorf("failed to save profilePic to disk: %w", err)
	}

	// we return the url path to retrieve it, not where it's stored on disk
	imgsrc := "/patients/" + patient.ID + "/profilepic/" + filename
	return imgsrc, nil
}

func ProfilePicPath(filename string) (string, error) {
	assetsDir := os.Getenv("ASSETS_DIR")
	if assetsDir == "" {
		return "", errors.New("failed to find assets directory")
	}

	path := filepath.Join(assetsDir, patientsDir)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.Mkdir(path, os.ModeDir|0755)
		if err != nil {
			return "", errors.New("failed to create assets/patients dir")
		}
	}

	return path + "/" + filename, nil
}
