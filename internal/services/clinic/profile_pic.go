package clinic

import (
	"errors"
	"fmt"
	"miconsul/internal/models"
	"path"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
)

var ErrProfilePicNotProvided = errors.New("profile picture not provided")

func SaveProfilePicToDisk(c fiber.Ctx, clinic models.Clinic) (string, error) {
	profilePic, err := c.FormFile("profilePic")
	if err != nil {
		return "", ErrProfilePicNotProvided
	}

	if clinic.UID == "" {
		return "", errors.New("can't save profile pic without clinic.UID")
	}

	filename := strings.TrimSpace(profilePic.Filename)
	filename = strings.ReplaceAll(filename, "\\", "/")
	filename = path.Base(filename)
	if filename == "" || filename == "." || filename == ".." {
		return "", errors.New("invalid profile picture filename")
	}

	profilePath := "/public/assets/profile_pics/" + clinic.UID + "_" + filename
	err = c.SaveFile(profilePic, "."+profilePath)
	if err != nil {
		return "", fmt.Errorf("save profilePic to disk: %w", err)
	}

	return profilePath, nil
}

// attachClinicPic saves the uploaded profile picture and stores its path on
// the clinic. It returns the clinic's picture path: the new one, or the
// current one when nothing was uploaded or saving failed, which is logged.
func (s service) attachClinicPic(c fiber.Ctx, userID uint, clinic models.Clinic) string {
	path, err := SaveProfilePicToDisk(c, clinic)
	if errors.Is(err, ErrProfilePicNotProvided) {
		return clinic.ProfilePic
	}
	if err != nil {
		log.Errorf("save clinic profile pic: %v", err)
		return clinic.ProfilePic
	}

	err = s.UpdateClinicProfilePic(c.Context(), userID, clinic.UID, path)
	if err != nil {
		log.Errorf("attach clinic profile pic: %v", err)
		return clinic.ProfilePic
	}

	return path
}
