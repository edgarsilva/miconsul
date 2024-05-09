package theme

import (
	"github.com/edgarsilva/go-scaffold/internal/view"

	"github.com/gofiber/fiber/v2"
)

func (s *service) HandleThemeChange(c *fiber.Ctx) error {
	theme := s.SessionUITheme(c)

	return view.Render(c, view.CmpThemeIcon(theme))
}
