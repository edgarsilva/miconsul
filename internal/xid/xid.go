// Package xid provides a XID value with a prefix and fixed length of 23 characters.
package xid

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/xid"
)

// Fixed consts used to validate the UID fields
const (
	alphabet = "0123456789abcdefghijklmnopqrstuv"
	length   = 23
)

func NewWithSB(prefix string) string {
	id := xid.New()
	return prefixID(prefix, id.String())
}

func New(prefix string) string {
	id := xid.New()
	return concatStr(prefix, id.String())
}

func prefixID(prefix, id string) string {
	if len(prefix) == 0 {
		return id
	}

	var sb strings.Builder
	sb.WriteString(prefix)
	sb.WriteString(id)

	return sb.String()
}

func concatStr(a, b string) string {
	if len(a) == 0 {
		return ""
	}

	return a + b
}

// Must is the same as New, but panics on error.
// func Must(suffix string) string { return nanoid.MustGenerate(alphabet, length) }

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
