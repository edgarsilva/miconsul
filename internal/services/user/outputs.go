package user

import "miconsul/internal/models"

type apiUsersOutput struct {
	Users []models.PublicUser
}

func newAPIUsersOutput(users []models.User) apiUsersOutput {
	publicUsers := make([]models.PublicUser, 0, len(users))
	for _, u := range users {
		publicUsers = append(publicUsers, u.Public())
	}

	return apiUsersOutput{Users: publicUsers}
}
