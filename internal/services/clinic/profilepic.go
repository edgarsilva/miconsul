package clinic

import (
	"errors"
	"fmt"
	"miconsul/internal/model"
	"path"
	"strings"

	"github.com/gofiber/fiber/v3"
)

var ErrProfilePicNotProvided = errors.New("profile picture not provided")

func SaveProfilePicToDisk(c fiber.Ctx, clinic model.Clinic) (string, error) {
	profilePic, err := c.FormFile("profilePic")
	if err != nil {
		return "", ErrProfilePicNotProvided
	}

	if clinic.ID == "" {
		return "", errors.New("can't save profile pic without clinic.ID")
	}

	filename := strings.TrimSpace(profilePic.Filename)
	filename = strings.ReplaceAll(filename, "\\", "/")
	filename = path.Base(filename)
	if filename == "" || filename == "." || filename == ".." {
		return "", errors.New("invalid profile picture filename")
	}

	profilePath := "/public/assets/profile_pics/" + clinic.ID + "_" + filename
	err = c.SaveFile(profilePic, "."+profilePath)
	if err != nil {
		return "", fmt.Errorf("failed to save profilePic to disk: %w", err)
	}

	return profilePath, nil
}
