package theme

import (
	"time"

	"miconsul/internal/lib/preferences"
	view "miconsul/internal/views"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/trace"
)

const themeCookieTTL = 365 * 24 * time.Hour

// HandleToggleTheme saves the chosen theme to the cookie and, for a signed-in
// user, to their record, then renders the theme button. The toggle posts the
// theme for each state (checked, unchecked) and, when checked, its box. The
// first choice made while the page follows the OS (from=os) reloads the page,
// since boosted swaps keep the old <html> and its missing data-theme.
// POST: /theme/toggle
func (s *service) HandleToggleTheme(c fiber.Ctx) error {
	state := "unchecked"
	if c.FormValue("theme", "") != "" {
		state = "checked"
	}

	theme := preferences.Theme(c.FormValue(state, ""))
	if theme == "" {
		theme = preferences.ThemeLight
	}

	c.Cookie(s.NewCookie("theme", theme, themeCookieTTL))

	if cu := s.CurrentUser(c); cu.ID != 0 {
		if err := s.SaveUserTheme(c.Context(), cu.ID, theme); err != nil {
			trace.SpanFromContext(c.Context()).RecordError(err)
		}
	}

	if c.FormValue("from", "") == "os" {
		c.Set("HX-Refresh", "true")
	}

	return view.Render(c, view.CmpThemeBtn(theme))
}
