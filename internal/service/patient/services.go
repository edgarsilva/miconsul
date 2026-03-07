package patient

import (
	"context"
	"errors"
	"fmt"
	"miconsul/internal/model"
	"miconsul/internal/server"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v3"
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

const patientsDir = "/patients"

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

func SaveProfilePicToDisk(c fiber.Ctx, patient model.Patient) (string, error) {
	profilePic, err := c.FormFile("profilePic")
	if err != nil {
		return "", profilePicFormFileErr(err)
	}

	if patient.ID == "" {
		return "", errors.New("failed to save profile pic without patient.ID")
	}

	originalFilename := strings.TrimSpace(profilePic.Filename)
	originalFilename = strings.ReplaceAll(originalFilename, "\\", "/")
	originalFilename = path.Base(originalFilename)
	safeFilename := cleanFilenameSegment(originalFilename)
	if safeFilename == "" || safeFilename == "." || safeFilename == ".." {
		return "", ErrInvalidFilename
	}

	filename := patient.ID + "_ppic_" + safeFilename
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
	if !isSafeFilename(filename) {
		return "", ErrInvalidFilename
	}

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

	return filepath.Join(path, filename), nil
}

func profilePicFormFileErr(err error) error {
	if isMissingProfilePicErr(err) {
		return ErrProfilePicNotProvided
	}

	return fmt.Errorf("failed to grab profilePic from form: %w", err)
}

func isMissingProfilePicErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, http.ErrMissingFile) {
		return true
	}

	errMsg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(errMsg, "no uploaded file") || strings.Contains(errMsg, "no such file")
}

func IsSafeProfilePicFilenameForPatient(patientID, filename string) bool {
	patientID = strings.TrimSpace(patientID)
	if patientID == "" || !isSafeFilename(filename) {
		return false
	}

	return strings.HasPrefix(filename, patientID+"_ppic_")
}

func isSafeFilename(filename string) bool {
	filename = strings.TrimSpace(filename)
	if filename == "" || filename == "." || filename == ".." {
		return false
	}
	if strings.ContainsAny(filename, "/\\") {
		return false
	}
	if strings.Contains(filename, "\x00") {
		return false
	}

	return cleanFilenameSegment(filename) == filename
}

func cleanFilenameSegment(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}

	b := strings.Builder{}
	b.Grow(len(v))
	for _, r := range v {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('_')
		}
	}

	return b.String()
}
