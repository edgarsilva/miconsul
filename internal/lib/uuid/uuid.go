// Package uuid generates prefixed UUIDv7 identifiers.
package uuid

import stduuid "uuid"

// New returns a UUIDv7 string with the given prefix.
func New(prefix string) string {
	return prefix + stduuid.NewV7().String()
}

// Pure returns a UUIDv7 string without a prefix.
func Pure() string {
	return stduuid.NewV7().String()
}
