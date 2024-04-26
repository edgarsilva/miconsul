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
	length   = 20
)

// New returns a new XID as a string
func New(prefix string) string {
	id := xid.New()
	return concatStr(prefix, id.String())
}

func concatStr(a, b string) string {
	if len(a) == 0 {
		return b
	}

	return a + b
}

// Validate checks if a given field name’s publicID (nanoid) value is valid according to
// the constraints defined by package publicid.
func Validate(prefix, id string) error {
	if id == "" {
		return errors.Errorf("XID cannot be blank")
	}

	if len(id) < length+len(prefix) {
		return errors.Errorf("XID should be %d characters long", length)
	}

	if strings.Trim(id, alphabet) != "" {
		return errors.Errorf("XID has invalid characters")
	}

	return nil
}
