package views

import (
	"miconsul/internal/lib/localize"
	"miconsul/internal/lib/preferences"
	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
)

// Toast describes the notification shown in the FE
type Toast struct {
	Msg   string
	Sub   string
	Level string
}

// Alert describes the inline alert a redirect carried to the page through a
// flash message.
type Alert struct {
	Msg   string
	Level string
}

type Ctx struct {
	fiber.Ctx
	Locale      string
	Theme       string
	Timeframe   string
	Toast       Toast
	Alert       Alert
	CurrentUser models.User
}

type ContextOption func(*Ctx)

var (
	phoneRegex = "^[\\+]?[(]?[0-9]{3}[)]?[-\\s\\.]?[0-9]{3}[-\\s\\.]?[0-9]{4,6}$"
	locales    = localize.New(preferences.LocaleSpanish, preferences.LocaleEnglish)
)

func (c *Ctx) l(key string) string {
	return locales.GetWithLocale(c.Locale, key)
}

func localeFromLocals(fc fiber.Ctx) string {
	locale, _ := fc.Locals("locale").(string)
	if locale = preferences.Locale(locale); locale == "" {
		return preferences.DefaultLocale
	}

	return locale
}

func themeFromLocals(fc fiber.Ctx) string {
	theme, _ := fc.Locals("theme").(string)
	return preferences.Theme(theme)
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

// NewCtx builds the view context from request locals and the flash toast and
// alert, then applies the options.
func NewCtx(fc fiber.Ctx, ctxOpts ...ContextOption) *Ctx {
	c := Ctx{
		Ctx:         fc,
		CurrentUser: currentUserFromLocals(fc),
		Locale:      localeFromLocals(fc),
		Theme:       themeFromLocals(fc),
		Toast: Toast{
			Msg:   fc.Redirect().Message("toast").Value,
			Level: fc.Redirect().Message("level").Value,
		},
		Alert: Alert{
			Msg:   fc.Redirect().Message("alert").Value,
			Level: fc.Redirect().Message("alert_level").Value,
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

// WithTheme sets the UI theme; values other than light|dark follow the OS setting.
func WithTheme(theme string) ContextOption {
	return func(c *Ctx) {
		c.Theme = preferences.Theme(theme)
	}
}

// WithLocale sets the locale; values outside the allowlist use the default.
func WithLocale(lang string) ContextOption {
	return func(c *Ctx) {
		if lang = preferences.Locale(lang); lang == "" {
			lang = preferences.DefaultLocale
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
