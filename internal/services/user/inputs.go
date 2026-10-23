package user

import (
	"miconsul/internal/lib/preferences"
	"miconsul/internal/models"
)

type userProfileUpdateInput struct {
	Name   string `form:"name"`
	Email  string `form:"email"`
	Phone  string `form:"phone"`
	Locale string `form:"locale"`
}

// toUserProfileUpdates maps the input to the user fields to update. A locale
// outside the allowlist is left empty, so the saved one stays.
func (in userProfileUpdateInput) toUserProfileUpdates() models.User {
	return models.User{
		Name:   in.Name,
		Email:  in.Email,
		Phone:  in.Phone,
		Locale: preferences.Locale(in.Locale),
	}
}
