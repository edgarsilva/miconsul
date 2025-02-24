// Package xid provides a XID value with a prefix and fixed length of 23 characters.
package xid

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/xid"
)

// Fixed consts used to validate the UID fields
const (
	avchabet = "0123456789abcdefghijklmnopqrstuv"
	length   = 20
)

// New returns a new XID as a string with a given prefix
func New(prefix string) string {
	id := xid.New()
	return prefix + id.String()
}

// New returns a new XID as a string
func Pure() string {
	id := xid.New()
	return id.String()
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

	if strings.Trim(id, avchabet) != "" {
		return errors.Errorf("XID has invalid characters")
	}

	return nil
}
