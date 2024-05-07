package view

import (
	"errors"
)

type CurrentUser interface {
	IsLoggedIn() bool
	ID() uint
	UID() string
	Email() string
	JWT() string
}

type layoutProps struct {
	CurrentUser
	Theme  string
	Locale string
}

func NewLayoutProps(props ...Prop) (layoutProps, error) {
	layoutProps := layoutProps{
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

type Prop func(layoutProps *layoutProps) error

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
