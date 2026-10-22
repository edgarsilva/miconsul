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
// user, to their record, then renders the theme button.
// POST: /theme/toggle
func (s *service) HandleToggleTheme(c fiber.Ctx) error {
	theme := preferences.ThemeLight
	if c.FormValue("theme", "") != "" {
		theme = preferences.ThemeDark
	}

	c.Cookie(s.NewCookie("theme", theme, themeCookieTTL))

	if cu := s.CurrentUser(c); cu.ID != 0 {
		if err := s.SaveUserTheme(c.Context(), cu.ID, theme); err != nil {
			trace.SpanFromContext(c.Context()).RecordError(err)
		}
	}

	return view.Render(c, view.CmpBtnTheme(theme))
}
