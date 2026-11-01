package views

import (
	"miconsul/internal/lib/localize"
	"miconsul/internal/lib/preferences"
	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
)

// Level is how a toast or alert is styled. Its only values are LevelInfo,
// LevelSuccess, LevelWarning and LevelError; the zero value is LevelInfo.
type Level struct{ name string }

var (
	LevelInfo    = Level{"info"}
	LevelSuccess = Level{"success"}
	LevelWarning = Level{"warning"}
	LevelError   = Level{"error"}
)

// String returns the level's name, as stored in flash messages.
func (l Level) String() string {
	if l.name == "" {
		return LevelInfo.name
	}

	return l.name
}

// Toast describes the notification shown in the FE
type Toast struct {
	Msg   string
	Sub   string
	Level Level
}

// Alert describes the inline alert a redirect carried to the page through a
// flash message.
type Alert struct {
	Msg   string
	Level Level
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
			Level: parseLevel(fc.Redirect().Message("level").Value),
		},
		Alert: Alert{
			Msg:   fc.Redirect().Message("alert").Value,
			Level: parseLevel(fc.Redirect().Message("alert_level").Value),
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
//   - toast: main text,
//   - sub: optional subtitle,
//   - level: how the toast is styled
func WithToast(toast, sub string, level Level) ContextOption {
	return func(c *Ctx) {
		c.Toast = Toast{
			Msg:   toast,
			Sub:   sub,
			Level: level,
		}
	}
}

// parseLevel returns the level named s, or LevelInfo when s names none.
func parseLevel(s string) Level {
	switch s {
	case LevelSuccess.name:
		return LevelSuccess
	case LevelWarning.name:
		return LevelWarning
	case LevelError.name:
		return LevelError
	default:
		return LevelInfo
	}
}
