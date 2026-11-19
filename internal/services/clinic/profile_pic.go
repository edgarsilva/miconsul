package clinic

import (
	"errors"
	"fmt"
	"miconsul/internal/models"
	"path"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
// current one when nothing was uploaded or saving failed, which is recorded on
// the span and logged.
func (s service) attachClinicPic(c fiber.Ctx, userID uint, clinic models.Clinic) string {
	ctx, span := s.Trace(c.Context(), "clinic/attachPic")
	defer span.End()

	span.SetAttributes(attribute.String("clinic.uid", clinic.UID))

	path, err := SaveProfilePicToDisk(c, clinic)
	if errors.Is(err, ErrProfilePicNotProvided) {
		return clinic.ProfilePic
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "save profile pic failed")
		log.Errorf("save clinic profile pic: %v", err)
		return clinic.ProfilePic
	}

	err = s.UpdateClinicProfilePic(ctx, userID, clinic.UID, path)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "attach profile pic failed")
		log.Errorf("attach clinic profile pic: %v", err)
		return clinic.ProfilePic
	}

	return path
}
