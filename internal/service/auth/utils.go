package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
)

// bodyParams extracts email and password from the request form values
func bodyParams(c *fiber.Ctx) (email, password string, err error) {
	email = c.FormValue("email", "")
	password = c.FormValue("password", "")
	if email == "" || password == "" {
		return email, password, fmt.Errorf("couldn't parse email or password from body: %q", err)
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
	c.Cookie(newCookie("Auth", "", time.Hour*0))
	c.Cookie(newCookie("JWT", "", time.Hour*0))
}
