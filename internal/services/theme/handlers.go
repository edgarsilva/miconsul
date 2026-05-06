package theme

import (
	"time"

	view "miconsul/internal/views"

	"github.com/gofiber/fiber/v3"
)

// HandleToggleTheme toggles and renders the active UI theme button.
// POST: /theme/toggle
func (s *service) HandleToggleTheme(c fiber.Ctx) error {
	theme := "light"
	if c.FormValue("theme", "") != "" {
		theme = "dark"
	}

	c.Cookie(s.NewCookie("theme", theme, 24*7*time.Hour))

	return view.Render(c, view.CmpBtnTheme(theme))
}
