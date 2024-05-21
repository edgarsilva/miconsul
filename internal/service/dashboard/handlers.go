package dashboard

import (
	"github.com/edgarsilva/go-scaffold/internal/view"

	"github.com/gofiber/fiber/v2"
)

// HandleBlogPage renders the blog home html page
//
// GET: /blog
func (s *service) HandleDashboardPage(c *fiber.Ctx) error {
	theme := s.SessionUITheme(c)

	cu, _ := s.CurrentUser(c)
	layoutProps, _ := view.NewLayoutProps(view.WithCurrentUser(cu), view.WithTheme(theme))
	return view.Render(c, view.DashboardPage(layoutProps))
}
