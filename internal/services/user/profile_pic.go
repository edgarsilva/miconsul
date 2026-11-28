package user

import (
	"context"
	"errors"
	"fmt"

	"miconsul/internal/lib/objstore"
	"miconsul/internal/lib/profilepic"
	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
)

const usersKeyPrefix = "users/"

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

// SaveProfilePic stores the uploaded profilePic in store as the user's avatar
// and returns the URL path that serves it.
func SaveProfilePic(c fiber.Ctx, store objstore.Store, user models.User) (string, error) {
	return saveUpload(c, store, user, "_avatar", "/profile/avatar")
}

// SaveProfilePicPreview stores the uploaded profilePic in store as the user's
// avatar preview and returns the URL path that serves it.
func SaveProfilePicPreview(c fiber.Ctx, store objstore.Store, user models.User) (string, error) {
	return saveUpload(c, store, user, "_preview", "/profile/avatar/preview")
}

// ProfilePicKey returns the object key of a user picture file.
func ProfilePicKey(filename string) string {
	return usersKeyPrefix + filename
}

// removeProfilePicFiles deletes the user's avatar and avatar preview.
func (s service) removeProfilePicFiles(ctx context.Context, userUID string) {
	for _, suffix := range []string{"_avatar", "_preview"} {
		err := s.ObjectStore.Delete(ctx, ProfilePicKey(userUID+suffix))
		if err != nil {
			log.Errorf("remove user profile pic file: %v", err)
		}
	}
}

func saveUpload(c fiber.Ctx, store objstore.Store, user models.User, suffix, urlPath string) (string, error) {
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

	err = profilepic.Put(c.Context(), store, ProfilePicKey(user.UID+suffix), upload)
	if err != nil {
		return "", fmt.Errorf("save profile pic: %w", err)
	}

	return urlPath, nil
}
