package home

import (
	"github.com/edgarsilva/go-scaffold/internal/view"

	"github.com/gofiber/fiber/v2"
)

func (s *service) HandleRoot(c *fiber.Ctx) error {
	theme := s.SessionGet(c, "theme", "cmky")
	props, _ := view.NewLayoutProps(view.WithTheme(theme))
	return view.Render(c, view.HomePage(props))
}
