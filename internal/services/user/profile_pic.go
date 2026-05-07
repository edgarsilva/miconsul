package user

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
)

const usersDir = "/users"

var ErrProfilePicNotProvided = errors.New("profile picture not provided")

func SaveProfilePicToDisk(c fiber.Ctx, user models.User, assetsDir string) (string, error) {
	profilePic, err := c.FormFile("profilePic")
	if err != nil {
		return "", profilePicFormFileErr(err)
	}

	if user.UID == "" {
		return "", errors.New("failed to save profile pic without user.UID")
	}

	filename := user.UID + "_avatar"

	filePath, err := ProfilePicPath(filename, assetsDir)
	if err != nil {
		return "", fmt.Errorf("failed to save profile pic without an ASSETS_DIR: %w", err)
	}

	err = c.SaveFile(profilePic, filePath)
	if err != nil {
		return "", fmt.Errorf("failed to save profilePic to disk: %w", err)
	}

	return "/profile/avatar", nil
}

func SaveProfilePicPreviewToTmp(c fiber.Ctx, user models.User, assetsDir string) (string, error) {
	profilePic, err := c.FormFile("profilePic")
	if err != nil {
		return "", profilePicFormFileErr(err)
	}

	if user.UID == "" {
		return "", errors.New("failed to save profile pic preview without user.UID")
	}

	filename := user.UID + "_preview"

	// Save tmp preview pic to /tmp directory
	filePath, err := ProfilePicPath(filename, assetsDir)
	if err != nil {
		return "", fmt.Errorf("failed to save profile pic without an ASSETS_DIR: %w", err)
	}

	err = c.SaveFile(profilePic, filePath)
	if err != nil {
		return "", fmt.Errorf("failed to save profile pic preview to tmp: %w", err)
	}

	return "/profile/avatar/preview", nil
}

func ProfilePicPath(filename, assetsDir string) (string, error) {
	if !isSafeFilename(filename) {
		return "", ErrInvalidFilename
	}

	assetsDir = strings.TrimSpace(assetsDir)
	if assetsDir == "" {
		return "", errors.New("failed to find assets directory")
	}

	absAssetsDir, err := filepath.Abs(assetsDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve assets directory: %w", err)
	}

	dirPath := filepath.Join(absAssetsDir, usersDir)
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return "", errors.New("failed to create assets/users dir")
	}

	return filepath.Join(dirPath, filename), nil
}

var ErrInvalidFilename = errors.New("invalid filename")

func IsSafeProfilePicFilenameForUser(userID, filename string) bool {
	userID = strings.TrimSpace(userID)
	if userID == "" || !isSafeFilename(filename) {
		return false
	}

	return strings.HasPrefix(filename, userID+"_ppic_")
}

func IsProfilePicPreviewFilenameForUser(userID, filename string) bool {
	userID = strings.TrimSpace(userID)
	if userID == "" || !isSafeFilename(filename) {
		return false
	}

	prefix := userID + "_profile_pic_preview"
	if filename == prefix {
		return true
	}

	return strings.HasPrefix(filename, prefix+".")
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
	return strings.Contains(errMsg, "no uploaded file") ||
		strings.Contains(errMsg, "no such file") ||
		strings.Contains(errMsg, "not multipart/form-data")
}

func isSafeFilename(filename string) bool {
	filename = strings.TrimSpace(filename)
	if filename == "" || filename == "." || filename == ".." {
		return false
	}
	if strings.Contains(filename, "..") {
		return false
	}
	if strings.Contains(filename, "../") {
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
