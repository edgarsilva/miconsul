package patient

import (
	"errors"
	"fmt"

	"github.com/edgarsilva/go-scaffold/internal/model"
	"github.com/edgarsilva/go-scaffold/internal/server"
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

func SaveProfilePicToDisk(c *fiber.Ctx, patient model.Patient) (string, error) {
	profilePic, err := c.FormFile("profilePic")
	if err != nil {
		return "", fmt.Errorf("failed to grab profilePic from form: %w", err)
	}

	if patient.ID == "" {
		return "", errors.New("can't save profile pic without patient.ID")
	}

	path := "/public/assets/profile_pics/" + patient.ID + "_" + profilePic.Filename
	err = c.SaveFile(profilePic, "."+path)
	if err != nil {
		return "", fmt.Errorf("failed to save profilePic to disk: %w", err)
	}

	return path, nil
}
