package auth

import (
	"crypto/rand"
	"encoding/hex"
	mrand "math/rand"
)

func newResetPasswordToken() (string, error) {
	return newHexToken(32)
}

func newConfirmEmailToken() string {
	token, err := newHexToken(32)
	if err != nil {
		return randomTokenRunes(32)
	}

	return token
}

func newHexToken(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

func randomTokenRunes(n int) string {
	letterRunes := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[mrand.Intn(len(letterRunes))]
	}

	return string(b)
}
