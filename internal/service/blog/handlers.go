package blog

import (
	"github.com/edgarsilva/go-scaffold/internal/view"

	"github.com/gofiber/fiber/v2"
)

// HandleBlogPage renders the blog home html page
//
// GET: /blog
func (s *service) HandleBlogPage(c *fiber.Ctx) error {
	theme := s.SessionUITheme(c)

	cu, _ := s.CurrentUser(c)
	vc, _ := view.NewCtx(c, view.WithCurrentUser(cu), view.WithTheme(theme))
	return view.Render(c, view.PageBlog(vc))
}
