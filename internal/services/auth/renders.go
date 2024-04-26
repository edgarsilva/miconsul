package auth

import (
	"github.com/edgarsilva/go-scaffold/internal/views"
	"github.com/gofiber/fiber/v2"
)

func (s *service) RenderLoginPage(c *fiber.Ctx, error error) error {
	cu, _ := s.CurrentUser(c)
	theme := s.SessionGet(c, "theme", "light")
	layoutProps, _ := views.NewLayoutProps(cu, views.WithTheme(theme))
	email := c.Query("email")

	return views.Render(c, views.LoginPage(email, error, layoutProps))
}
