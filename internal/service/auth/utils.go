package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	mrand "math/rand"

	"github.com/gofiber/fiber/v2"
)

// authParams extracts email and password from the request form values
func authParams(c *fiber.Ctx) (email, password string, err error) {
	email = c.FormValue("email", "")
	password = c.FormValue("password", "")
	if email == "" || password == "" {
		err = errors.New("email and password can't be blank")
		return email, password, err
	}

	return email, password, nil
}

// resetPasswordEmailParam returns the email address string from either
// FormValue, URLParam or Query in that order of existance
func resetPasswordEmailParam(c *fiber.Ctx) (string, error) {
	email := c.FormValue("email", "")
	if email != "" {
		return email, nil
	}

	email = c.Params("email", "")
	if email != "" {
		return email, nil
	}

	email = c.Query("email", "")
	if email != "" {
		return email, nil
	}

	err := errors.New("email can't be blank")
	return "", err
}

func resetPasswordToken() (string, error) {
	return randHexToken(32)
}

func randToken() string {
	token, err := randHexToken(32)
	if err != nil {
		return randStringRunes(32)
	}

	return token
}

func randHexToken(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func randStringRunes(n int) string {
	letterRunes := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[mrand.Intn(len(letterRunes))]
	}

	return string(b)
}
