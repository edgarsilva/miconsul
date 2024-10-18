package clinic

import (
	"errors"
	"fmt"
	"miconsul/internal/model"
	"miconsul/internal/server"

	"github.com/gofiber/fiber/v2"
)

type service struct {
	*server.Server
}

func NewService(s *server.Server) service {
	return service{
		Server: s,
	}
}

func (s service) TakeClinicByID(c *fiber.Ctx, id string) (model.Clinic, error) {
	cu, _ := s.CurrentUser(c)
	clinic := model.Clinic{
		ID:     id,
		UserID: cu.ID,
	}

	result := s.DB.
		Model(&clinic).
		Where("user_id = ?", cu.ID).
		Take(&clinic)
	if result.Error != nil {
		return model.Clinic{}, result.Error
	}

	return clinic, nil
}

func (s service) FindClinicsByTerm(c *fiber.Ctx, term string) ([]model.Clinic, error) {
	cu, _ := s.CurrentUser(c)
	clinics := []model.Clinic{}

	s.DB.WithContext(c.UserContext()).
		Model(cu).
		Scopes(model.GlobalFTS(term)).
		Limit(QUERY_LIMIT).
		Association("Clinics").
		Find(&clinics)

	return clinics, nil
}

func SaveProfilePicToDisk(c *fiber.Ctx, clinic model.Clinic) (string, error) {
	profilePic, err := c.FormFile("profilePic")
	if err != nil {
		return "", fmt.Errorf("failed to grab profilePic from form: %w", err)
	}

	if clinic.ID == "" {
		return "", errors.New("can't save profile pic without patient.ID")
	}

	path := "/public/assets/profile_pics/" + clinic.ID + "_" + profilePic.Filename
	err = c.SaveFile(profilePic, "."+path)
	if err != nil {
		return "", fmt.Errorf("failed to save profilePic to disk: %w", err)
	}

	return path, nil
}
