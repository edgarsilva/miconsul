package auth

import "crypto/rand"

func newResetPasswordToken() string {
	return newSecretToken()
}

func newConfirmEmailToken() string {
	return newSecretToken()
}

// newSecretToken returns a random token for links sent to users.
func newSecretToken() string {
	return rand.Text()
}
