// Package dashboard provides the dashboard handlers for the web app UI
package dashboard

import (
	"errors"

	"miconsul/internal/lib/libtime"
	"miconsul/internal/models"
	view "miconsul/internal/views"

	"github.com/a-h/templ"
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
	timeframe := libtime.ParseTimeframe(c.Query("timeframe", ""))
	statuses := models.ParseAppointmentStatuses(c.Query("status", ""))

	appointments, appointmentsErr := s.FindAppointmentsBy(c.Context(), cu.ID, timeframe, statuses)
	clinic, clinicErr := s.FavoriteClinic(c, cu.ID)
	stats, statsErr := s.CalcDashboardStats(c.Context(), cu)
	feedEvents, feedErr := s.FindFeedEventsByUserID(c.Context(), cu.ID, 10)

	vc := view.NewCtx(c)
	if err := errors.Join(appointmentsErr, clinicErr, statsErr, feedErr); err != nil {
		log.Error("load dashboard:", err)
		vc = view.NewCtx(c, view.WithToast("Failed to load part of the dashboard, please try again", "", view.LevelError))
	}

	return view.Render(c, view.DashboardPage(vc, stats, appointments, clinic, feedEvents))
}

// HandleAppointmentsSection renders the dashboard appointments section for
// its timeframe and status filter links.
// GET: /dashboard/appointments
func (s *service) HandleAppointmentsSection(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	timeframe := libtime.ParseTimeframe(c.Query("timeframe", ""))
	statuses := models.ParseAppointmentStatuses(c.Query("status", ""))

	appointments, err := s.FindAppointmentsBy(c.Context(), cu.ID, timeframe, statuses)
	vc := view.NewCtx(c)
	section := view.DashboardAppointmentsSection(vc, appointments)
	if err != nil {
		log.Error("load dashboard appointments:", err)
		return view.Render(c, templ.Join(
			section,
			view.CmpToastPartial("Failed to load appointments, please try again", "", view.LevelError),
		))
	}

	return view.Render(c, section)
}
