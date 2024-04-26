package home

import (
	"github.com/edgarsilva/go-scaffold/internal/views"

	"github.com/gofiber/fiber/v2"
)

func (s *service) HandleRoot(c *fiber.Ctx) error {
	theme := s.SessionGet(c, "theme", "cmky")
	cu, _ := s.CurrentUser(c)
	props, _ := views.NewLayoutProps(cu, views.WithTheme(theme))
	return views.Render(c, views.HomePage(props))
}
