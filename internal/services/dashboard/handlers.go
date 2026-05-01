// Package dashboard provides the dashboard handlers for the web app UI
package dashboard

import (
	"time"

	"miconsul/internal/lib/libtime"
	"miconsul/internal/models"
	view "miconsul/internal/views"

	"github.com/gofiber/fiber/v3"
)

// HandleDashboardPage renders the home dashboard
// GET: /
// GET: /dashboard
func (s *service) HandleDashboardPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	appointments := []models.Appointment{}
	query := s.DB.WithContext(c.Context()).Model(models.Appointment{}).Where("user_id = ?", cu.ID)

	timeframe := c.Query("timeframe", "")
	switch timeframe {
	case "day":
		query.Scopes(models.AppointmentBookedToday)
	case "week":
		query.Scopes(models.AppointmentBookedThisWeek)
	case "month":
		query.Scopes(models.AppointmentBookedThisMonth)
	default:
		query.Where("booked_at > ?", libtime.BoD(time.Now()))
	}

	clinic, _ := s.FavoriteClinic(c, cu.ID)
	vc, _ := view.NewCtx(c)
	stats := s.CalcDashboardStats(c.Context(), cu)

	return view.Render(c, view.DashboardPage(vc, stats, appointments, clinic))
}
