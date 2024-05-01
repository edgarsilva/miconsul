package auth

import (
	"github.com/edgarsilva/go-scaffold/internal/view"
	"github.com/gofiber/fiber/v2"
)

func (s *service) RenderLoginPage(c *fiber.Ctx, error error) error {
	theme := s.SessionGet(c, "theme", "light")
	layoutProps, _ := view.NewLayoutProps(view.WithTheme(theme))
	email := c.Query("email")

	return view.Render(c, view.LoginPage(email, error, layoutProps))
}
