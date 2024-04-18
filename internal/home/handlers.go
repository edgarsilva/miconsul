package handlers

import (
	"fiber-blueprint/internal/views"

	"github.com/gofiber/fiber/v2"
)

func (s *service) HandleRoot(c *fiber.Ctx) error {
	th := s.SessionGet(c, "theme", "cmky")
	return views.Render(c, LandingPage(th))
}
