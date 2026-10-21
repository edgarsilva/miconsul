package middleware

import (
	"github.com/gofiber/fiber/v3"
)

// UITheme sets the request's UI theme (light|dark) from the theme cookie.
func UITheme() fiber.Handler {
	return func(c fiber.Ctx) error {
		theme := "light"
		if c.Cookies("theme") == "dark" {
			theme = "dark"
		}
		c.Locals("theme", theme)

		return c.Next()
	}
}
