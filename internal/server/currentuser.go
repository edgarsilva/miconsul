package server

import "github.com/edgarsilva/go-scaffold/internal/database"

type currentUser struct {
	*database.User
	token string
}

func (cu currentUser) IsLoggedIn() bool {
	if cu.User == nil {
		return false
	}

	return cu.User.ID != 0
}

func (cu currentUser) Email() string {
	if cu.User == nil {
		return ""
	}

	return cu.User.Email
}

func (cu currentUser) ID() uint {
	if cu.User == nil {
		return 0
	}

	return cu.User.ID
}

func (cu currentUser) UID() string {
	if cu.User == nil {
		return ""
	}

	return cu.User.UID
}

func (cu currentUser) JWT() string {
	if cu.User == nil {
		return ""
	}

	return cu.token
}
