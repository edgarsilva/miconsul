package theme

import (
	"miconsul/internal/lib/handlerutils"
	"miconsul/internal/view"
	"time"

	"github.com/gofiber/fiber/v2"
)

func (s *service) HandleChangeUITheme(c *fiber.Ctx) error {
	theme := s.SessionUITheme(c)

	if theme == "light" {
		theme = "dark"
	} else {
		theme = "light"
	}

	c.Cookie(handlerutils.NewCookie("theme", theme, 24*time.Hour*31))

	return view.Render(c, view.CmpBtnTheme(theme))
}
