package middleware

import (
	"github.com/gofiber/fiber/v2"
)

func UITheme() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		theme := c.Query("theme", "")
		if theme == "" {
			theme = c.Cookies("theme", "")
		}

		if theme == "" {
			theme = "light"
		}

		c.Cookies("theme", theme)
		c.Locals("theme", theme)

		return c.Next()
	}
}
