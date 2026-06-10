package middleware

import (
	"github.com/gofiber/fiber/v3"
)

func UITheme() fiber.Handler {
	return func(c fiber.Ctx) error {
		theme := c.Cookies("theme", "light")
		c.Locals("theme", theme)

		return c.Next()
	}
}
