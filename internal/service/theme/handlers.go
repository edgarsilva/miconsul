package theme

import (
	"github.com/edgarsilva/go-scaffold/internal/view"

	"github.com/gofiber/fiber/v2"
)

func (s *service) HandleThemeChange(c *fiber.Ctx) error {
	theme := c.Query("theme", "night")

	if theme == "light" {
		s.SessionSet(c, "theme", "light")
	} else {
		s.SessionSet(c, "theme", "dark")
	}

	// return c.SendStatus(fiber.StatusNoContent)
	return view.Render(c, view.ThemeIcon(theme))
}
