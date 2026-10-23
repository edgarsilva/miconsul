package auth

import (
	"crypto/rand"
	"encoding/hex"
)

func newResetPasswordToken() (string, error) {
	return newHexToken(32)
}

func newConfirmEmailToken() string {
	return rand.Text()
}

func newHexToken(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}
