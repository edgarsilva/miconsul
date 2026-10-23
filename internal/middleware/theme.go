package middleware

import (
	"miconsul/internal/lib/preferences"

	"github.com/gofiber/fiber/v3"
)

// UITheme sets the request's UI theme from the theme cookie: light, dark, or
// "" to follow the OS setting.
func UITheme() fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Locals("theme", preferences.Theme(c.Cookies("theme")))

		return c.Next()
	}
}
