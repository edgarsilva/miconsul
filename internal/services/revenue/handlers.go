// Package revenue provides the revenue page handlers.
package revenue

import (
	"time"

	view "miconsul/internal/views"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
)

// HandleIndexPage renders the user's revenue for the last 12 months and the
// selected month (?month=YYYY-MM, the current month by default) by clinic.
// GET: /revenue
func (s *service) HandleIndexPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	now := time.Now()
	selected, _ := ParseMonth(c.Query("month", ""))

	report, err := s.Report(c.Context(), cu.ID, now, selected)
	if err != nil {
		log.Error("load revenue report:", err)
		vc := view.NewCtx(c, view.WithToast("Failed to load revenue, please try again", "", view.LevelError))
		return view.Render(c, view.RevenueIndexPage(vc, report))
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.RevenueIndexPage(vc, report))
}
