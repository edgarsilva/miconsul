package views

import (
	"miconsul/internal/lib/localize"
	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
)

// Toast describes the notification shown in the FE
type Toast struct {
	Msg   string
	Sub   string
	Level string
}

type Ctx struct {
	fiber.Ctx
	Locale      string
	Theme       string
	Timeframe   string
	Toast       Toast
	CurrentUser models.User
}

type ContextOption func(*Ctx)

var (
	phoneRegex = "^[\\+]?[(]?[0-9]{3}[)]?[-\\s\\.]?[0-9]{3}[-\\s\\.]?[0-9]{4,6}$"
	locales    = localize.New("es-MX", "en-US")
)

func (c *Ctx) l(key string) string {
	return locales.GetWithLocale(c.Locale, key)
}

func localeFromLocals(fc fiber.Ctx) string {
	iloc := fc.Locals("locale")
	loc, ok := iloc.(string)
	if !ok {
		loc = "es-MX"
	}

	return loc
}

func themeFromLocals(fc fiber.Ctx) string {
	itheme := fc.Locals("theme")
	theme, ok := itheme.(string)
	if !ok {
		theme = "light"
	}

	return theme
}

func currentUserFromLocals(fc fiber.Ctx) models.User {
	iuser := fc.Locals("current_user")
	cu, ok := iuser.(models.User)
	if !ok {
		return models.User{}
	}

	return cu
}

func CU(fc fiber.Ctx) models.User {
	return currentUserFromLocals(fc)
}

// NewCtx builds the view context from request locals and the query, then
// applies the options.
func NewCtx(fc fiber.Ctx, ctxOpts ...ContextOption) *Ctx {
	c := Ctx{
		Ctx:         fc,
		CurrentUser: currentUserFromLocals(fc),
		Locale:      localeFromLocals(fc),
		Theme:       themeFromLocals(fc),
		Toast: Toast{
			Msg:   fc.Query("toast", ""),
			Sub:   fc.Query("sub", ""),
			Level: fc.Query("level", ""),
		},
	}

	for _, opt := range ctxOpts {
		opt(&c)
	}

	return &c
}

func WithCurrentUser(cu models.User) ContextOption {
	return func(c *Ctx) {
		c.CurrentUser = cu
	}
}

// WithTheme sets the UI theme; values other than light|dark fall back to light.
func WithTheme(theme string) ContextOption {
	return func(c *Ctx) {
		c.Theme = "light"
		if theme == "dark" {
			c.Theme = "dark"
		}
	}
}

func WithLocale(lang string) ContextOption {
	return func(c *Ctx) {
		if lang == "" {
			lang = "es-MX"
		}

		c.Locale = lang
	}
}

// WithToast sets up a notification toast to be rendered by the base layout
// views
//
// # Note: you can also pass thist 3 params as querpyParams in redirects
//
//   - toast: main text,
//   - sub: optional subtitle,
//   - level: alert level, one of success, error, warning or info (defaults to info)
func WithToast(toast, sub, level string) ContextOption {
	return func(c *Ctx) {
		c.Toast = Toast{
			Msg:   toast,
			Sub:   sub,
			Level: level,
		}
	}
}
