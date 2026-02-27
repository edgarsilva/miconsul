package dashboard

import (
	"miconsul/internal/lib/libtime"
	"miconsul/internal/model"
	"miconsul/internal/view"
	"time"

	"github.com/gofiber/fiber/v3"
)

// HandleDashboardPage renders the home dashboard
//
//	GET: / or /dashboard
func (s *service) HandleDashboardPage(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect().To("/login")
	}

	appointments := []model.Appointment{}
	query := s.DB.WithContext(c.Context()).Model(model.Appointment{}).Where("user_id = ?", cu.ID)

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

	query.
		Preload("Clinic").
		Preload("Patient").
		Limit(10).
		Find(&appointments)

	clinic := model.Clinic{UserID: cu.ID, Favorite: true}
	s.DB.WithContext(c.Context()).Model(&model.Clinic{}).Where(clinic, "UserID", "favorite").Order("created_at").Take(&clinic)

	if clinic.ID == "" {
		s.DB.WithContext(c.Context()).Model(&model.Clinic{}).Where(clinic, "UserID").Order("created_at").Take(&clinic)
	}

	vc, _ := view.NewCtx(c)
	stats := s.CalcDashboardStats(c.Context(), cu)

	return view.Render(c, view.DashboardPage(vc, stats, appointments, clinic))
}
