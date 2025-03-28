package patient

import (
	"errors"
	"fmt"
	"miconsul/internal/model"
	"miconsul/internal/server"
	"os"
	"path/filepath"

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

func SaveProfilePicToDisk(c *fiber.Ctx, patient model.Patient) (string, error) {
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
