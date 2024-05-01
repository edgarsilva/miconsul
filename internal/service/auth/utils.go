package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
)

// AuthParams extracts email and password from the request body
func bodyParams(c *fiber.Ctx) (email, password string, err error) {
	type params struct {
		Email    string `json:"name" xml:"name" form:"email"`
		Password string `json:"password" xml:"pass" form:"password"`
	}

	p := params{}
	if err := c.BodyParser(&p); err != nil {
		return "", "", fmt.Errorf("couldn't parse email or password from body: %q", err)
	}

	email = p.Email
	password = p.Password

	return email, password, nil
}

// newCookie creates a new cookie and returns a pointer to the cookie
func newCookie(name, value string, validFor time.Duration) *fiber.Cookie {
	return &fiber.Cookie{
		Name:    name,
		Value:   value,
		Expires: time.Now().Add(validFor),
		// MaxAge:   60 * 5,
		Secure:   os.Getenv("env") == "production",
		HTTPOnly: true,
	}
}

func invalidateCookies(c *fiber.Ctx) {
	c.Cookie(newCookie("Auth", "", time.Hour*24))
	c.Cookie(newCookie("JWT", "", time.Hour*24))
}
