// Package profilepic names, validates and stores uploaded profile pictures.
package profilepic

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"miconsul/internal/lib/objstore"
)

var (
	ErrNotProvided     = errors.New("profile picture not provided")
	ErrUnsupportedType = errors.New("unsupported picture type")
)

// Accept lists the picture MIME types for the accept attribute of a file
// input. Phones offer the camera and gallery for them, and desktop pickers show
// every extension of these types.
const Accept = "image/jpeg,image/png,image/webp"

var contentTypes = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".jpe":  "image/jpeg",
	".jfif": "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
}

// FormFileError maps the error of a multipart file lookup. A missing file
// becomes ErrNotProvided.
func FormFileError(err error) error {
	if err == nil {
		return nil
	}
	if isMissing(err) {
		return ErrNotProvided
	}

	return fmt.Errorf("grab profilePic from form: %w", err)
}

// RandomName returns prefix, 16 random bytes in hex and the extension of
// uploadName, for example ("u1_ppic_", "me.PNG") -> "u1_ppic_9f2c...e1.png".
// It returns ErrUnsupportedType when the extension is not a picture type.
func RandomName(prefix, uploadName string) (string, error) {
	ext, err := uploadExt(uploadName)
	if err != nil {
		return "", err
	}

	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate profile pic name: %w", err)
	}

	return prefix + hex.EncodeToString(buf) + ext, nil
}

// CheckType returns ErrUnsupportedType when uploadName does not have a picture
// extension.
func CheckType(uploadName string) error {
	_, err := uploadExt(uploadName)
	return err
}

// ContentType returns the MIME type for the extension of uploadName. It returns
// ErrUnsupportedType when the extension is not a picture type.
func ContentType(uploadName string) (string, error) {
	ext, err := uploadExt(uploadName)
	if err != nil {
		return "", err
	}

	return contentTypes[ext], nil
}

// Put stores the uploaded picture in store under key, with the content type of
// the upload's extension.
func Put(ctx context.Context, store objstore.Store, key string, upload *multipart.FileHeader) error {
	contentType, err := ContentType(upload.Filename)
	if err != nil {
		return err
	}

	file, err := upload.Open()
	if err != nil {
		return fmt.Errorf("open uploaded picture: %w", err)
	}

	err = store.Put(ctx, key, file, upload.Size, contentType)
	closeErr := file.Close()
	if err != nil {
		return fmt.Errorf("store picture: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("close uploaded picture: %w", closeErr)
	}

	return nil
}

// IsSafeFilename reports whether name is a single path segment made of
// letters, digits, dot, dash and underscore.
func IsSafeFilename(name string) bool {
	if name == "" || name == "." || name == ".." || len(name) > 255 {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.', r == '-', r == '_':
		default:
			return false
		}
	}

	return true
}

func uploadExt(uploadName string) (string, error) {
	uploadName = strings.ReplaceAll(strings.TrimSpace(uploadName), "\\", "/")
	ext := strings.ToLower(filepath.Ext(filepath.Base(uploadName)))
	if _, ok := contentTypes[ext]; !ok {
		return "", ErrUnsupportedType
	}

	return ext, nil
}

func isMissing(err error) bool {
	if errors.Is(err, http.ErrMissingFile) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no uploaded file") ||
		strings.Contains(msg, "no such file") ||
		strings.Contains(msg, "not multipart/form-data")
}
