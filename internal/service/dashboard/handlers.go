package dashboard

import (
	"time"

	"github.com/edgarsilva/go-scaffold/internal/model"
	"github.com/edgarsilva/go-scaffold/internal/util"
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
	query := s.DB.Model(model.Appointment{}).Where("user_id = ?", cu.ID)

	timeframe := c.Query("timeframe", "")
	switch timeframe {
	case "day":
		query.Scopes(model.AppointmentsBookedToday)
	case "week":
		query.Scopes(model.AppointmentsBookedThisWeek)
	case "month":
		query.Scopes(model.AppointmentsBookedThisMonth)
	default:
		query.Where("booked_at > ?", util.BoD(time.Now()))
	}

	query.Preload("Clinic").
		Preload("Patient").
		Limit(10).
		Find(&appointments)

	theme := s.SessionUITheme(c)
	lp, _ := view.NewLayoutProps(c, view.WithCurrentUser(cu), view.WithTheme(theme))
	return view.Render(c, view.DashboardPage(appointments, lp))
}
