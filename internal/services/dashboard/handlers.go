// Package dashboard provides the dashboard handlers for the web app UI
package dashboard

import (
	"errors"
	"strings"

	view "miconsul/internal/views"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
)

// HandleHomePage renders the public landing page.
// GET: /
func (s *service) HandleHomePage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	if cu.IsLoggedIn() {
		return s.Redirect(c, "/dashboard?timeframe=day")
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.LandingPage(vc))
}

// HandleDashboardPage renders the home dashboard
// GET: /dashboard
func (s *service) HandleDashboardPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	timeframe := c.Query("timeframe", "day")
	statusFilter := strings.TrimSpace(c.Query("status", ""))

	appointments, appointmentsErr := s.FindAppointmentsBy(c.Context(), cu.ID, timeframe, statusFilter, 10)
	clinic, clinicErr := s.FavoriteClinic(c, cu.ID)
	stats, statsErr := s.CalcDashboardStats(c.Context(), cu)
	feedEvents, feedErr := s.FindFeedEventsByUserID(c.Context(), cu.ID, 10)

	vc := view.NewCtx(c)
	if err := errors.Join(appointmentsErr, clinicErr, statsErr, feedErr); err != nil {
		log.Error("load dashboard:", err)
		vc = view.NewCtx(c, view.WithToast("Failed to load part of the dashboard, please try again", "", "error"))
	}

	return view.Render(c, view.DashboardPage(vc, stats, appointments, clinic, feedEvents))
}
