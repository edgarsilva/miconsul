package patient

import (
	"errors"
	"fmt"
	"strings"

	"miconsul/internal/lib/profilepic"
	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const patientsDir = "/patients"

// SaveProfilePicToDisk stores the uploaded profilePic under a random name and
// returns the URL path that serves it.
func SaveProfilePicToDisk(c fiber.Ctx, patient models.Patient, assetsDir string) (string, error) {
	upload, err := c.FormFile("profilePic")
	if err != nil {
		return "", profilepic.FormFileError(err)
	}

	if patient.UID == "" {
		return "", errors.New("save profile pic without patient.UID")
	}

	filename, err := profilepic.RandomName(patient.UID+"_ppic_", upload.Filename)
	if err != nil {
		return "", fmt.Errorf("name profile pic: %w", err)
	}

	path, err := ProfilePicPath(filename, assetsDir)
	if err != nil {
		return "", fmt.Errorf("locate profile pic: %w", err)
	}

	err = c.SaveFile(upload, path)
	if err != nil {
		return "", fmt.Errorf("save profilePic to disk: %w", err)
	}

	return "/patients/" + patient.UID + "/profilepic/" + filename, nil
}

// attachPatientPic saves the uploaded profile picture and stores its path on
// the patient. It returns the patient's picture path: the new one, or the
// current one when nothing was uploaded or saving failed, which is recorded on
// the span and logged.
func (s service) attachPatientPic(c fiber.Ctx, userID uint, patient models.Patient) string {
	ctx, span := s.Trace(c.Context(), "patient/attachPic")
	defer span.End()

	span.SetAttributes(attribute.String("patient.uid", patient.UID))

	path, err := SaveProfilePicToDisk(c, patient, s.Env.AssetsDir)
	if errors.Is(err, profilepic.ErrNotProvided) {
		return patient.ProfilePic
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "save profile pic failed")
		log.Errorf("save patient profile pic: %v", err)
		return patient.ProfilePic
	}

	err = s.UpdatePatientProfilePic(ctx, userID, patient.UID, path)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "attach profile pic failed")
		log.Errorf("attach patient profile pic: %v", err)
		return patient.ProfilePic
	}

	return path
}

// ProfilePicPath returns the file path of a patient picture in assetsDir.
func ProfilePicPath(filename, assetsDir string) (string, error) {
	return profilepic.Path(assetsDir, patientsDir, filename)
}

// IsSafeProfilePicFilenameForPatient reports whether filename is a safe
// picture name that belongs to patientID.
func IsSafeProfilePicFilenameForPatient(patientID, filename string) bool {
	patientID = strings.TrimSpace(patientID)
	if patientID == "" || !profilepic.IsSafeFilename(filename) {
		return false
	}

	return strings.HasPrefix(filename, patientID+"_ppic_")
}
