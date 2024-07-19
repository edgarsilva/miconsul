package handlerutils

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
)

// newCookie creates a new cookie and returns a pointer to the cookie
func NewCookie(name, value string, validFor time.Duration) *fiber.Cookie {
	return &fiber.Cookie{
		Name:     name,
		Value:    value,
		Expires:  time.Now().Add(validFor),
		Secure:   os.Getenv("env") == "production",
		HTTPOnly: true,
	}
}

// invalidateSessionCookies blanks session cookies and expires them
// time.Hour*0
func InvalidateCookies(c *fiber.Ctx, cookieNames ...string) {
	c.ClearCookie(cookieNames...)
	for _, name := range cookieNames {
		c.Cookie(NewCookie(name, "", time.Hour*0))
	}
}
