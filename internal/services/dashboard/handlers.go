package dashboard

import (
	"miconsul/internal/lib/libtime"
	"miconsul/internal/models"
	view "miconsul/internal/views"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// HandleDashboardPage renders the home dashboard
// GET: /
// GET: /dashboard
func (s *service) HandleDashboardPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

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

	clinic := model.Clinic{}
	clinic, _ = gorm.G[model.Clinic](s.DB.GormDB()).
		Where("user_id = ? AND favorite = ?", cu.ID, true).
		Order("created_at").
		Take(c.Context())

	if clinic.ID == "" {
		clinic, _ = gorm.G[model.Clinic](s.DB.GormDB()).
			Where("user_id = ?", cu.ID).
			Order("created_at").
			Take(c.Context())
	}

	vc, _ := view.NewCtx(c)
	stats := s.CalcDashboardStats(c.Context(), cu)

	return view.Render(c, view.DashboardPage(vc, stats, appointments, clinic))
}
