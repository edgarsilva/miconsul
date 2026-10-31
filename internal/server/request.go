package server

import (
	"miconsul/internal/lib/preferences"
	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
)

// CurrentUser returns request-scoped user from fiber locals only.
// Auth/session resolution is handled by auth.Authenticate + middleware binding.
func (s *Server) CurrentUser(c fiber.Ctx) models.User {
	userIface := c.Locals("current_user")
	cu, ok := userIface.(models.User)
	if !ok {
		return models.User{}
	}

	return cu
}

// CurrentTheme returns the UI theme set by the theme middleware: light, dark,
// or "" to follow the OS setting.
func (s *Server) CurrentTheme(c fiber.Ctx) string {
	theme, _ := c.Locals("theme").(string)
	return preferences.Theme(theme)
}

// CurrentLocale returns the locale set by the locale middleware, or the
// default locale.
func (s *Server) CurrentLocale(c fiber.Ctx) string {
	locale, _ := c.Locals("locale").(string)
	if locale = preferences.Locale(locale); locale == "" {
		return preferences.DefaultLocale
	}

	return locale
}

// IsHTMX returns true if the request was initiated by HTMX.
func (s *Server) IsHTMX(c fiber.Ctx) bool {
	isHTMX := c.Get("HX-Request", "")
	return isHTMX == "true"
}
