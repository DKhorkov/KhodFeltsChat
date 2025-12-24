package mappers

import (
	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	"github.com/DKhorkov/kfc/internal/domains"
)

func MapUser(user domains.User) schemas.User {
	return schemas.User{
		ID:             user.ID,
		Username:       user.Username,
		Email:          user.Email,
		EmailConfirmed: user.EmailConfirmed,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
	}
}

func MapUsers(users []domains.User) []schemas.User {
	result := make([]schemas.User, len(users))
	for i, user := range users {
		result[i] = MapUser(user)
	}

	return result
}
