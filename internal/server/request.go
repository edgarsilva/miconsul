package server

import (
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

// CurrentTheme returns the UI theme (light|dark) set by the theme middleware.
func (s *Server) CurrentTheme(c fiber.Ctx) string {
	theme, ok := c.Locals("theme").(string)
	if !ok || theme == "" {
		return "light"
	}

	return theme
}

// CurrentLocale returns the locale set by the locale middleware.
func (s *Server) CurrentLocale(c fiber.Ctx) string {
	locale, ok := c.Locals("locale").(string)
	if !ok || locale == "" {
		return "es-MX"
	}

	return locale
}

// IsHTMX returns true if the request was initiated by HTMX.
func (s *Server) IsHTMX(c fiber.Ctx) bool {
	isHTMX := c.Get("HX-Request", "")
	return isHTMX == "true"
}

// NotHTMX returns true if the request was not initiated by HTMX.
func (s *Server) NotHTMX(c fiber.Ctx) bool {
	return !s.IsHTMX(c)
}
