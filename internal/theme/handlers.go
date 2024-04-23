package theme

import (
	"rtx-blog/internal/views"

	"github.com/gofiber/fiber/v2"
)

func (s *service) HandleThemeChange(c *fiber.Ctx) error {
	theme := c.Query("theme", "light")

	if theme == "light" {
		s.SessionSet(c, "theme", "light")
	} else {
		s.SessionSet(c, "theme", "dark")
	}

	// return c.SendStatus(fiber.StatusNoContent)
	return views.Render(c, views.ThemeIcon(theme))
}
