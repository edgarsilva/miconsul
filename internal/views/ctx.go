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

func (ctx *Ctx) l(key string) string {
	return l(ctx.Locale, key)
}

func l(lang, key string) string {
	return locales.GetWithLocale(lang, key)
}

func localeFromLocals(c fiber.Ctx) string {
	iloc := c.Locals("locale")
	loc, ok := iloc.(string)
	if !ok {
		loc = "es-MX"
	}

	return loc
}

func themeFromLocals(c fiber.Ctx) string {
	itheme := c.Locals("theme")
	theme, ok := itheme.(string)
	if !ok {
		theme = "light"
	}

	return theme
}

func currentUserFromLocals(c fiber.Ctx) models.User {
	iuser := c.Locals("current_user")
	cu, ok := iuser.(models.User)
	if !ok {
		return models.User{}
	}

	return cu
}

func CU(c fiber.Ctx) models.User {
	return currentUserFromLocals(c)
}

// NewCtx builds the view context from request locals and the query, then
// applies the options.
func NewCtx(c fiber.Ctx, ctxOpts ...ContextOption) *Ctx {
	ctx := Ctx{
		Ctx:         c,
		CurrentUser: currentUserFromLocals(c),
		Locale:      localeFromLocals(c),
		Theme:       themeFromLocals(c),
		Toast: Toast{
			Msg:   c.Query("toast", ""),
			Sub:   c.Query("sub", ""),
			Level: c.Query("level", ""),
		},
	}

	for _, opt := range ctxOpts {
		opt(&ctx)
	}

	return &ctx
}

func WithCurrentUser(cu models.User) ContextOption {
	return func(props *Ctx) {
		props.CurrentUser = cu
	}
}

// WithTheme sets the UI theme; values other than light|dark fall back to light.
func WithTheme(theme string) ContextOption {
	return func(props *Ctx) {
		props.Theme = "light"
		if theme == "dark" {
			props.Theme = "dark"
		}
	}
}

func WithLocale(lang string) ContextOption {
	return func(props *Ctx) {
		if lang == "" {
			lang = "es-MX"
		}

		props.Locale = lang
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
	return func(props *Ctx) {
		props.Toast = Toast{
			Msg:   toast,
			Sub:   sub,
			Level: level,
		}
	}
}
