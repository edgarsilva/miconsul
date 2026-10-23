package middleware

import (
	"miconsul/internal/lib/preferences"

	"github.com/gofiber/fiber/v3"
)

// LocaleLang defines a universal middleware to stract Locale lang en-US, es-MX, etc
func LocaleLang() fiber.Handler {
	return func(c fiber.Ctx) error {
		lang := ""

		switch c.AcceptsLanguages(preferences.LocaleEnglish, preferences.LocaleSpanish, "es-US", "en", "es") {
		case preferences.LocaleEnglish, "en":
			lang = preferences.LocaleEnglish
		default:
			lang = preferences.DefaultLocale
		}

		c.Locals("locale", lang)

		return c.Next()
	}
}
