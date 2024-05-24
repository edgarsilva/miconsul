package dashboard

import (
	"github.com/edgarsilva/go-scaffold/internal/model"
	"github.com/edgarsilva/go-scaffold/internal/view"

	"github.com/gofiber/fiber/v2"
)

// HandleBlogPage renders the blog home html page
//
// GET: /blog
func (s *service) HandleDashboardPage(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	appointments := []model.Appointment{}
	s.DB.Model(&cu).
		Preload("Clinic").
		Preload("Patient").
		Association("Appointments").
		Find(&appointments)

	theme := s.SessionUITheme(c)
	lp, _ := view.NewLayoutProps(c, view.WithCurrentUser(cu), view.WithTheme(theme))
	return view.Render(c, view.DashboardPage(appointments, lp))
}
