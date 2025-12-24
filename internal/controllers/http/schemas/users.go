package schemas

import "time"

// User represents a user's contact record.
// swagger:model
type User struct {
	// Full name of the user.
	// required: true
	// nullable: false
	// minimum: 1
	ID uint64 `json:"id"`

	// Unique username of the user.
	// required: true
	// nullable: false
	// minLength: 5
	// maxLength: 70
	// example: D3M0S
	Username string `json:"username"`

	// Email of the user.
	// required: true
	// nullable: false
	// format: email
	// example: alexqwerty@yandex.ru
	Email string `json:"email"`

	// Represents whether email of the user confirmed or not.
	// required: true
	// nullable: false
	EmailConfirmed bool `json:"emailConfirmed"`

	// Represents datetime when user was registered.
	// required: true
	// nullable: false
	// format: date-time
	CreatedAt time.Time `json:"createdAt"`

	// Represents datetime when user was updated.
	// required: true
	// nullable: false
	// format: date-time
	UpdatedAt time.Time `json:"updatedAt"`
}

// IDInput
// swagger:parameters IDInput
type IDInput struct {
	// Unique identifier
	// required: true
	// nullable: false
	// in: path
	ID int `json:"id"`
}

// GetUserByIDInput
// swagger:parameters GetUserByID
type GetUserByIDInput struct {
	IDInput
}

// GetUsersInput
// swagger:parameters GetUsers
type GetUsersInput struct {
	Pagination

	// Username or it's part to search users that matches it
	// required: false
	// nullable: false
	// in: query
	Username string `json:"username"`
}

// UpdateUserInput
// swagger:parameters UpdateCurrentUser
type UpdateUserInput struct {
	// Information about current User for update
	// required: true
	// nullable: false
	// in: body
	Body struct {
		UsernameInput
	}
}
