package middleware

import (
	"github.com/gofiber/fiber/v2"
)

func UITheme() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		theme := c.Cookies("theme", "light")
		c.Locals("theme", theme)

		return c.Next()
	}
}
