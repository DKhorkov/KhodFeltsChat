package domains

import "time"

type User struct {
	ID             uint64    `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	EmailConfirmed bool      `json:"emailConfirmed"`
	Password       string    `json:"password"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type RawUpdateUserDTO struct {
	AccessToken string `json:"accessToken"`
	Username    string `json:"username"`
}

type UpdateUserDTO struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
}

type UsersFilters struct {
	Username *string `json:"username,omitempty"`
}
