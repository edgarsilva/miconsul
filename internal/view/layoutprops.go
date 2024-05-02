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
			return errors.New("current user must exist, you are passing 'nil' or an emtpy interface")
		}

		props.CurrentUser = cu

		return nil
	}
}

func WithTheme(theme string) Prop {
	return func(props *layoutProps) error {
		if theme == "" {
			return errors.New("theme(light|dark) must be set)")
		}

		props.Theme = theme

		return nil
	}
}
