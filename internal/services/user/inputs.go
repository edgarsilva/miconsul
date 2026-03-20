package user

import "miconsul/internal/models"

type userProfileUpdateInput struct {
	Name  string `form:"name"`
	Email string `form:"email"`
	Phone string `form:"phone"`
}

func (in userProfileUpdateInput) toUserProfileUpdates() model.User {
	return model.User{
		Name:  in.Name,
		Email: in.Email,
		Phone: in.Phone,
	}
}
