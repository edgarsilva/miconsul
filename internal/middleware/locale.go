package middleware

import (
	"github.com/gofiber/fiber/v3"
)

// LocaleLang defines a universal middleware to stract Locale lang en-US, es-MX, etc
func LocaleLang() func(c fiber.Ctx) error {
	return func(c fiber.Ctx) error {
		lang := ""

		switch c.AcceptsLanguages("en-US", "es-MX", "es-US", "en", "es") {
		case "es-US", "es-MX", "es":
			lang = "es-MX"
		case "en-US", "en":
			lang = "en-US"
		default:
			lang = "es-MX"
		}

		c.Locals("locale", lang)

		return c.Next()
	}
}
