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
	Theme     string
	Locale    string
	Flash     string
	Toast     Toast
	Timeframe string
	Path      string
	Query     map[string]string
}

type Prop func(layoutProps *layoutProps) error

var (
	phoneRegex = "^[\\+]?[(]?[0-9]{3}[)]?[-\\s\\.]?[0-9]{3}[-\\s\\.]?[0-9]{4,6}$"
	locales    = localize.New("es-MX", "en-US")
)

func NewLayoutProps(c *fiber.Ctx, props ...Prop) (layoutProps, error) {
	layoutProps := layoutProps{
		CurrentUser: new(DummyUser),
		Query:       c.Queries(),
		Path:        c.Path(),
		Locale:      "es-MX",
		Theme:       "light",
		Toast:       Toast{},
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

func WithToast(msg, sub string) Prop {
	return func(props *layoutProps) error {
		props.Toast = Toast{
			Msg: msg,
			Sub: sub,
		}

		return nil
	}
}

func WithFlash(flash string) Prop {
	return func(props *layoutProps) error {
		props.Flash = flash

		return nil
	}
}
