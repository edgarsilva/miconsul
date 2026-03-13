package theme

import (
	"miconsul/internal/views"
	"time"

	"github.com/gofiber/fiber/v3"
)

// HandleToggleTheme toggles and renders the active UI theme button.
// POST: /theme/toggle
func (s *service) HandleToggleTheme(c fiber.Ctx) error {
	theme := s.SessionUITheme(c)
	theme = c.Params("theme", theme)

	c.Cookie(s.NewCookie("theme", theme, 24*7*time.Hour))

	return view.Render(c, view.CmpBtnTheme(theme))
}
