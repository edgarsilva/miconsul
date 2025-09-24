package theme

import (
	"miconsul/internal/lib/handlerutils"
	"miconsul/internal/view"
	"time"

	"github.com/gofiber/fiber/v2"
)

func (s *service) HandleToggleTheme(c *fiber.Ctx) error {
	theme := s.SessionUITheme(c)
	theme = c.Params("theme", theme)

	c.Cookie(handlerutils.NewCookie("theme", theme, 24*7*time.Hour))

	return view.Render(c, view.CmpBtnTheme(theme))
}
