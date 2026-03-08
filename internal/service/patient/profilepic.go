package patient

import (
	"errors"
	"fmt"
	"miconsul/internal/model"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v3"
)

const patientsDir = "/patients"

func SaveProfilePicToDisk(c fiber.Ctx, patient model.Patient, assetsDir string) (string, error) {
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
	path, err := ProfilePicPath(filename, assetsDir)
	if err != nil {
		return "", fmt.Errorf("failed to save profile pic without an ASSETS_DIR: %w", err)
	}

	err = c.SaveFile(profilePic, path)
	if err != nil {
		return "", fmt.Errorf("failed to save profilePic to disk: %w", err)
	}

	imgsrc := "/patients/" + patient.ID + "/profilepic/" + filename
	return imgsrc, nil
}

func ProfilePicPath(filename, assetsDir string) (string, error) {
	if !isSafeFilename(filename) {
		return "", ErrInvalidFilename
	}

	assetsDir = strings.TrimSpace(assetsDir)
	if assetsDir == "" {
		return "", errors.New("failed to find assets directory")
	}

	path := filepath.Join(assetsDir, patientsDir)
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", errors.New("failed to create assets/patients dir")
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
