package clinic

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"miconsul/internal/lib/objstore"
	"miconsul/internal/lib/profilepic"
	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const clinicsKeyPrefix = "clinics/"

const unsupportedPicMessage = "Unsupported picture type, use JPG, PNG or WebP"

// hasUnsupportedProfilePic reports whether the request uploads a profilePic
// that is not a picture type.
func hasUnsupportedProfilePic(c fiber.Ctx) bool {
	upload, err := c.FormFile("profilePic")
	if err != nil || upload == nil {
		return false
	}

	return errors.Is(profilepic.CheckType(upload.Filename), profilepic.ErrUnsupportedType)
}

// SaveProfilePic stores the uploaded profilePic in store under a random name
// and returns the URL path that serves it.
func SaveProfilePic(c fiber.Ctx, store objstore.Store, clinic models.Clinic) (string, error) {
	upload, err := c.FormFile("profilePic")
	if err != nil {
		return "", profilepic.FormFileError(err)
	}

	if clinic.UID == "" {
		return "", errors.New("save profile pic without clinic.UID")
	}

	filename, err := profilepic.RandomName(clinic.UID+"_ppic_", upload.Filename)
	if err != nil {
		return "", fmt.Errorf("name profile pic: %w", err)
	}

	err = profilepic.Put(c.Context(), store, ProfilePicKey(filename), upload)
	if err != nil {
		return "", fmt.Errorf("save profile pic: %w", err)
	}

	return "/clinics/" + clinic.UID + "/profilepic/" + filename, nil
}

// ProfilePicKey returns the object key of a clinic picture file.
func ProfilePicKey(filename string) string {
	return clinicsKeyPrefix + filename
}

// IsSafeProfilePicFilenameForClinic reports whether filename is a safe
// picture name that belongs to clinicID.
func IsSafeProfilePicFilenameForClinic(clinicID, filename string) bool {
	clinicID = strings.TrimSpace(clinicID)
	if clinicID == "" || !profilepic.IsSafeFilename(filename) {
		return false
	}

	return strings.HasPrefix(filename, clinicID+"_ppic_")
}

// attachClinicPic saves the uploaded profile picture and stores its path on
// the clinic. It returns the clinic's picture path: the new one, or the
// current one when nothing was uploaded or saving failed, which is recorded on
// the span and logged.
func (s service) attachClinicPic(c fiber.Ctx, userID uint, clinic models.Clinic) string {
	ctx, span := s.Trace(c.Context(), "clinic/attachPic")
	defer span.End()

	span.SetAttributes(attribute.String("clinic.uid", clinic.UID))

	path, err := SaveProfilePic(c, s.Objects, clinic)
	if errors.Is(err, profilepic.ErrNotProvided) {
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

	s.removeReplacedPic(ctx, clinic, path)

	return path
}

// removeReplacedPic deletes the clinic's previous picture file after a new one
// is saved. Pictures that are not stored in the object store are kept.
func (s service) removeReplacedPic(ctx context.Context, clinic models.Clinic, newPath string) {
	prefix := "/clinics/" + clinic.UID + "/profilepic/"
	if clinic.ProfilePic == newPath || !strings.HasPrefix(clinic.ProfilePic, prefix) {
		return
	}

	filename := strings.TrimPrefix(clinic.ProfilePic, prefix)
	if !IsSafeProfilePicFilenameForClinic(clinic.UID, filename) {
		return
	}

	err := s.Objects.Delete(ctx, ProfilePicKey(filename))
	if err != nil {
		log.Errorf("remove replaced clinic profile pic: %v", err)
	}
}
