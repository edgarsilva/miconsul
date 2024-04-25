package views

import "github.com/edgarsilva/go-scaffold/internal/database"

type Props struct {
	Theme       string
	Locale      string
	Counter     string
	CurrentUser database.User
}
