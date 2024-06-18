package theme

import (
	"miconsul/internal/view"

	"github.com/gofiber/fiber/v2"
)

func (s *service) HandleThemeChange(c *fiber.Ctx) error {
	theme := s.SessionUITheme(c)

	return view.Render(c, view.CmpBtnTheme(theme))
}
