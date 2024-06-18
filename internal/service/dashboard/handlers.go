package dashboard

import (
	"time"

	"miconsul/internal/common"
	"miconsul/internal/model"
	"miconsul/internal/view"

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
		query.Scopes(model.AppointmentBookedToday)
	case "week":
		query.Scopes(model.AppointmentBookedThisWeek)
	case "month":
		query.Scopes(model.AppointmentBookedThisMonth)
	default:
		query.Where("booked_at > ?", common.BoD(time.Now()))
	}

	query.Preload("Clinic").
		Preload("Patient").
		Limit(10).
		Find(&appointments)

	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithCurrentUser(cu), view.WithTheme(theme))
	stats := s.CalcDashboardStats()
	return view.Render(c, view.DashboardPage(vc, stats, appointments))
}
