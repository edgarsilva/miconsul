package view

import (
	"errors"

	"miconsul/internal/localize"
	"github.com/gofiber/fiber/v2"
)

type CurrentUser interface {
	IsLoggedIn() bool
}

// Toast notification required
type Toast struct {
	Msg   string
	Sub   string
	Level string
}

type Ctx struct {
	*fiber.Ctx
	CurrentUser
	Locale    string
	Theme     string
	Timeframe string
	Toast     Toast
}

type Prop func(ctx *Ctx) error

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

func NewCtx(c *fiber.Ctx, props ...Prop) (*Ctx, error) {
	locI := c.Locals("locale")
	loc, ok := locI.(string)
	if !ok {
		loc = "es-MX"
	}

	ctx := Ctx{
		Ctx:         c,
		CurrentUser: new(DummyUser),
		Locale:      loc,
		Theme:       "light",
		Toast: Toast{
			Msg:   c.Query("toast", ""),
			Sub:   c.Query("sub", ""),
			Level: c.Query("level", ""),
		},
	}

	for _, prop := range props {
		err := prop(&ctx)
		if err != nil {
			return &ctx, nil
		}
	}

	return &ctx, nil
}

func WithCurrentUser(cu CurrentUser) Prop {
	return func(props *Ctx) error {
		if cu == nil {
			return errors.New("current user must exist, you might be passing an emtpy(nil) interface")
		}

		props.CurrentUser = cu

		return nil
	}
}

func WithTheme(theme string) Prop {
	return func(props *Ctx) error {
		if theme == "" {
			return errors.New("theme can't be blank if you are trying to set it)")
		}

		if theme != "light" && theme != "dark" {
			return errors.New("theme must be either light or dark)")
		}

		props.Theme = theme

		return nil
	}
}

func WithLocale(lang string) Prop {
	return func(props *Ctx) error {
		if lang == "" {
			lang = "es-MX"
		}

		props.Locale = lang

		return nil
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
func WithToast(toast, sub, level string) Prop {
	return func(props *Ctx) error {
		props.Toast = Toast{
			Msg:   toast,
			Sub:   sub,
			Level: level,
		}

		return nil
	}
}
