package blog

import (
	"github.com/edgarsilva/go-scaffold/internal/view"

	"github.com/gofiber/fiber/v2"
)

func (s *service) HandleRoot(c *fiber.Ctx) error {
	theme := c.Query("theme", "")
	if theme == "" {
		theme = s.SessionGet(c, "theme", "")
	}
	props, _ := view.NewLayoutProps(view.WithTheme(theme))
	return view.Render(c, view.BlogPage(props))
}
