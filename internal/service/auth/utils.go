package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"time"

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

// newCookie creates a new cookie and returns a pointer to the cookie
func newCookie(name, value string, validFor time.Duration) *fiber.Cookie {
	return &fiber.Cookie{
		Name:     name,
		Value:    value,
		Expires:  time.Now().Add(validFor),
		Secure:   os.Getenv("env") == "production",
		HTTPOnly: true,
	}
}

// invalidateCookies sets Auth & JWT cookies to blank "" and expires them
// time.Hour*0
func invalidateCookies(c *fiber.Ctx) {
	c.ClearCookie("Auth", "JWT")
	c.Cookie(newCookie("Auth", "", time.Hour*0))
	c.Cookie(newCookie("JWT", "", time.Hour*0))
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

func resetPasswordGenToken() (string, error) {
	return randHexToken(32)
}

func randHexToken(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
