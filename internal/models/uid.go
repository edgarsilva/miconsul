package models

import (
	"encoding/base32"
	"errors"
	"uuid"
)

var uidEncoding = base32.NewEncoding("0123456789abcdefghjkmnpqrstvwxyz").WithPadding(base32.NoPadding)

// newUID returns a new UUIDv7 as 26 lowercase base32 characters. The alphabet
// leaves out i, l, o and u and is in byte order, so UIDs sort by creation time.
func newUID() string {
	id := uuid.NewV7()
	return uidEncoding.EncodeToString(id[:])
}

// parseUID returns the UUID that a UID built by newUID encodes.
func parseUID(uid string) (uuid.UUID, error) {
	b, err := uidEncoding.DecodeString(uid)
	if err != nil {
		return uuid.UUID{}, err
	}

	if len(b) != len(uuid.UUID{}) {
		return uuid.UUID{}, errors.New("uid has the wrong length")
	}

	return uuid.UUID(b), nil
}
