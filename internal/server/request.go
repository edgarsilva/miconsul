package server

import (
	"miconsul/internal/model"

	"github.com/gofiber/fiber/v3"
)

// CurrentUser returns request-scoped user from fiber locals only.
// Auth/session resolution is handled by auth.Authenticate + middleware binding.
func (s *Server) CurrentUser(c fiber.Ctx) model.User {
	userIface := c.Locals("current_user")
	cu, ok := userIface.(model.User)
	if !ok {
		return model.User{}
	}

	return cu
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
