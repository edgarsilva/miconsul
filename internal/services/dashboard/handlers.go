// Package dashboard provides the dashboard handlers for the web app UI
package dashboard

import (
	"strings"

	"miconsul/internal/models"
	view "miconsul/internal/views"

	"github.com/gofiber/fiber/v3"
)

// HandleHomePage renders the public landing page.
// GET: /
func (s *service) HandleHomePage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	if cu.IsLoggedIn() {
		return s.Redirect(c, "/dashboard?timeframe=day")
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.LandingPage(vc))
}

// HandleDashboardPage renders the home dashboard
// GET: /dashboard
func (s *service) HandleDashboardPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	timeframe := c.Query("timeframe", "day")
	statusFilter := strings.TrimSpace(c.Query("status", ""))

	appointments := []models.Appointment{}
	query := s.DB.WithContext(c.Context()).
		Model(models.Appointment{}).
		Preload("Patient").
		Preload("Clinic").
		Where("user_id = ?", cu.ID)

	switch timeframe {
	case "day":
		query.Scopes(models.AppointmentBookedToday)
	case "week":
		query.Scopes(models.AppointmentBookedThisWeek)
	case "month":
		query.Scopes(models.AppointmentBookedThisMonth)
	default:
		query.Scopes(models.AppointmentBookedToday)
	}

	if statusFilter != "" {
		statuses := strings.Split(statusFilter, ",")
		for i := range statuses {
			statuses[i] = strings.TrimSpace(statuses[i])
		}
		query.Where("status IN ?", statuses)
	}

	_ = query.Limit(10).Find(&appointments).Error

	clinic, _ := s.FavoriteClinic(c, cu.ID)
	vc, _ := view.NewCtx(c)
	stats := s.CalcDashboardStats(c.Context(), cu)

	feedEvents, _ := s.FindFeedEventsByUserID(c.Context(), cu.ID, 10)

	return view.Render(c, view.DashboardPage(vc, stats, appointments, clinic, feedEvents))
}
