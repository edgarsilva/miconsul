package user

import (
	"errors"
	"fmt"

	"miconsul/internal/lib/profilepic"
	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
)

const usersDir = "/users"

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

// SaveProfilePicToDisk stores the uploaded profilePic as the user's avatar and
// returns the URL path that serves it.
func SaveProfilePicToDisk(c fiber.Ctx, user models.User, assetsDir string) (string, error) {
	return saveUpload(c, user, assetsDir, "_avatar", "/profile/avatar")
}

// SaveProfilePicPreviewToTmp stores the uploaded profilePic as the user's
// avatar preview and returns the URL path that serves it.
func SaveProfilePicPreviewToTmp(c fiber.Ctx, user models.User, assetsDir string) (string, error) {
	return saveUpload(c, user, assetsDir, "_preview", "/profile/avatar/preview")
}

// ProfilePicPath returns the file path of a user picture in assetsDir.
func ProfilePicPath(filename, assetsDir string) (string, error) {
	return profilepic.Path(assetsDir, usersDir, filename)
}

func saveUpload(c fiber.Ctx, user models.User, assetsDir, suffix, urlPath string) (string, error) {
	upload, err := c.FormFile("profilePic")
	if err != nil {
		return "", profilepic.FormFileError(err)
	}

	if user.UID == "" {
		return "", errors.New("save profile pic without user.UID")
	}

	err = profilepic.CheckType(upload.Filename)
	if err != nil {
		return "", fmt.Errorf("check profile pic: %w", err)
	}

	filePath, err := ProfilePicPath(user.UID+suffix, assetsDir)
	if err != nil {
		return "", fmt.Errorf("locate profile pic: %w", err)
	}

	err = c.SaveFile(upload, filePath)
	if err != nil {
		return "", fmt.Errorf("save profile pic to disk: %w", err)
	}

	return urlPath, nil
}
