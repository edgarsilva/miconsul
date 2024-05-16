package view

import (
	"errors"

	"github.com/edgarsilva/go-scaffold/internal/localize"
)

type CurrentUser interface {
	IsLoggedIn() bool
}

type layoutProps struct {
	CurrentUser
	Theme  string
	Locale string
}

type Prop func(layoutProps *layoutProps) error

var locales = localize.New("es-MX", "en-US")

func NewLayoutProps(props ...Prop) (layoutProps, error) {
	layoutProps := layoutProps{
		Locale:      "es-MX",
		Theme:       "light",
		CurrentUser: new(DummyUser),
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
