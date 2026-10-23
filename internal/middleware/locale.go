package middleware

import (
	"miconsul/internal/lib/preferences"

	"github.com/gofiber/fiber/v3"
)

// LocaleLang sets the request's locale from the locale cookie or, without
// one, from Accept-Language.
func LocaleLang() fiber.Handler {
	return func(c fiber.Ctx) error {
		lang := preferences.Locale(c.Cookies("locale"))
		if lang == "" {
			lang = acceptedLocale(c)
		}

		c.Locals("locale", lang)

		return c.Next()
	}
}

// acceptedLocale maps Accept-Language to an allowed locale. The default comes
// first, so a missing header or "*" picks it.
func acceptedLocale(c fiber.Ctx) string {
	switch c.AcceptsLanguages(preferences.LocaleSpanish, "es-US", "es", preferences.LocaleEnglish, "en") {
	case preferences.LocaleEnglish, "en":
		return preferences.LocaleEnglish
	default:
		return preferences.DefaultLocale
	}
}
