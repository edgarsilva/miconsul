package view

import (
	"errors"

	"github.com/edgarsilva/go-scaffold/internal/localize"
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

type layoutProps struct {
	CurrentUser
	*fiber.Ctx
	Locale    string
	Theme     string
	Timeframe string
	Toast     Toast
}

type Prop func(layoutProps *layoutProps) error

var (
	phoneRegex = "^[\\+]?[(]?[0-9]{3}[)]?[-\\s\\.]?[0-9]{3}[-\\s\\.]?[0-9]{4,6}$"
	locales    = localize.New("es-MX", "en-US")
)

func NewLayoutProps(c *fiber.Ctx, props ...Prop) (layoutProps, error) {
	layoutProps := layoutProps{
		Ctx:         c,
		CurrentUser: new(DummyUser),
		Locale:      "es-MX",
		Theme:       "light",
		Toast: Toast{
			Msg:   c.Query("toast", ""),
			Sub:   c.Query("sub", ""),
			Level: c.Query("level", ""),
		},
	}

	for _, prop := range props {
		err := prop(&layoutProps)
		if err != nil {
			return layoutProps, nil
		}
	}

	return layoutProps, nil
}

func l(lang, key string) string {
	return locales.GetWithLocale(lang, key)
}

func WithCurrentUser(cu CurrentUser) Prop {
	return func(props *layoutProps) error {
		if cu == nil {
			return errors.New("current user must exist, you might be passing an emtpy(nil) interface")
		}

		props.CurrentUser = cu

		return nil
	}
}

func WithTheme(theme string) Prop {
	return func(props *layoutProps) error {
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
	return func(props *layoutProps) error {
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
	return func(props *layoutProps) error {
		props.Toast = Toast{
			Msg:   toast,
			Sub:   sub,
			Level: level,
		}

		return nil
	}
}
