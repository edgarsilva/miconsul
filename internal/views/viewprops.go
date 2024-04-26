package views

import "errors"

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

func NewLayoutProps(currentUser CurrentUser, props ...Prop) (layoutProps, error) {
	layoutProps := layoutProps{
		Theme:       "light",
		CurrentUser: currentUser,
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

func WithTheme(theme string) Prop {
	return func(props *layoutProps) error {
		if theme == "" {
			return errors.New("theme(light|dark) must be set)")
		}

		props.Theme = theme

		return nil
	}
}
