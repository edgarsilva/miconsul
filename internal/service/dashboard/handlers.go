package dashboard

import (
	"miconsul/internal/lib/libtime"
	"miconsul/internal/model"
	"miconsul/internal/view"
	"time"

	"github.com/gofiber/fiber/v2"
)

// HandleDashboardPage renders the home dashboard
//
//	GET: / or /dashboard
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
		query.Where("booked_at > ?", libtime.BoD(time.Now()))
	}

	clinic := model.Clinic{UserID: cu.ID, Favorite: true}
	s.DB.Where(clinic, "UserID", "favorite").Order("created_at").Take(&clinic)

	if clinic.ID == "" {
		s.DB.Where(clinic, "UserID").Order("created_at").Take(&clinic)
	}

	query.
		Preload("Clinic").
		Preload("Patient").
		Limit(10).
		Find(&appointments)

	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithCurrentUser(cu), view.WithTheme(theme))
	stats := s.CalcDashboardStats(cu)
	return view.Render(c, view.DashboardPage(vc, stats, appointments, clinic))
}
