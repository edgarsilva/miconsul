package home

import (
	"github.com/edgarsilva/go-scaffold/internal/views"

	"github.com/gofiber/fiber/v2"
)

func (s *service) HandleRoot(c *fiber.Ctx) error {
	theme := s.SessionGet(c, "theme", "cmky")
	return views.Render(c, HomePage(views.Props{
		Theme: theme,
	}))
}
