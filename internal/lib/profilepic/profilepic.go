// Package profilepic names, validates and locates uploaded profile pictures on disk.
package profilepic

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrNotProvided     = errors.New("profile picture not provided")
	ErrInvalidFilename = errors.New("invalid filename")
	ErrUnsupportedType = errors.New("unsupported picture type")
	ErrNoAssetsDir     = errors.New("assets directory not set")
)

// Accept lists the picture extensions for the accept attribute of a file input.
const Accept = ".jpg,.jpeg,.png,.gif,.webp,.avif,.heic"

var allowedExts = map[string]struct{}{
	".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".webp": {}, ".avif": {}, ".heic": {},
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

// Path returns the file path of filename in assetsDir/subdir and creates the
// directory. It returns ErrInvalidFilename when filename is not safe.
func Path(assetsDir, subdir, filename string) (string, error) {
	if !IsSafeFilename(filename) {
		return "", ErrInvalidFilename
	}

	assetsDir = strings.TrimSpace(assetsDir)
	if assetsDir == "" {
		return "", ErrNoAssetsDir
	}

	absDir, err := filepath.Abs(assetsDir)
	if err != nil {
		return "", fmt.Errorf("resolve assets directory: %w", err)
	}

	dir := filepath.Join(absDir, subdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s directory: %w", subdir, err)
	}

	return filepath.Join(dir, filename), nil
}

// Remove deletes filename from assetsDir/subdir. A file that does not exist is
// not an error. It returns ErrInvalidFilename when filename is not safe.
func Remove(assetsDir, subdir, filename string) error {
	if !IsSafeFilename(filename) {
		return ErrInvalidFilename
	}

	assetsDir = strings.TrimSpace(assetsDir)
	if assetsDir == "" {
		return ErrNoAssetsDir
	}

	absDir, err := filepath.Abs(assetsDir)
	if err != nil {
		return fmt.Errorf("resolve assets directory: %w", err)
	}

	err = os.Remove(filepath.Join(absDir, subdir, filename))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove profile pic: %w", err)
	}

	return nil
}

func uploadExt(uploadName string) (string, error) {
	uploadName = strings.ReplaceAll(strings.TrimSpace(uploadName), "\\", "/")
	ext := strings.ToLower(filepath.Ext(filepath.Base(uploadName)))
	if _, ok := allowedExts[ext]; !ok {
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
