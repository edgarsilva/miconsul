package auth

import "miconsul/internal/models"

type showUserOutput struct {
	User models.PublicUser
}

func newShowUserOutput(u models.User) showUserOutput {
	return showUserOutput{User: u.Public()}
}
