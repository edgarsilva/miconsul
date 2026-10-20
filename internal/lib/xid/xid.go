// Package xid provides a XID value with a prefix and fixed length of 23 characters.
package xid

import (
	"github.com/rs/xid"
)

// New returns a new XID as a string with a given prefix
func New(prefix string) string {
	id := xid.New()
	return prefix + id.String()
}

// Pure returns a new XID as a string without prefix
func Pure() string {
	id := xid.New()
	return id.String()
}
