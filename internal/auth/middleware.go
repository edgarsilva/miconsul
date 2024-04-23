package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

func OnlyAuthenticated(c *fiber.Ctx) error {
	log.Info("Inside Auth Middleware")

	type Cookies struct {
		Auth string `cookie:"Auth"`
		JWT  string `cookie:"JWT"`
	}

	cookies := Cookies{}
	if err := c.CookieParser(&cookies); err != nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	if cookies.Auth == "" && cookies.JWT == "" {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	c.Locals("userID", cookies.Auth)
	c.Locals("JWT", cookies.JWT)

	return c.Next()
}
