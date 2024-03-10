// Package nanoid provides a nano ID value with a suffix and fixed length.
package nanoid

import (
	"strings"

	nanoid "github.com/matoous/go-nanoid/v2"
	"github.com/pkg/errors"
)

// Fixed nanoid parameters used in the Rails application.
const (
	alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	length   = 12
)

// New generates a unique nanoid ID.
func New(suffix string) (string, error) {
	var sb strings.Builder

	if len(suffix) > 0 {
		sb.WriteString(suffix)
	}

	id, err := nanoid.Generate(alphabet, length)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate nanoid")
	}

	sb.WriteString(id)
	return sb.String(), nil
}

// Must is the same as New, but panics on error.
func Must(suffix string) string { return nanoid.MustGenerate(alphabet, length) }

// Validate checks if a given field name’s publicID (nanoid) value is valid according to
// the constraints defined by package publicid.
func Validate(fieldName, id string) error {
	if id == "" {
		return errors.Errorf("%s cannot be blank", fieldName)
	}

	if len(id) < length {
		return errors.Errorf("%s should be %d characters long", fieldName, length)
	}

	if strings.Trim(id, alphabet) != "" {
		return errors.Errorf("%s has invalid characters", fieldName)
	}

	return nil
}
