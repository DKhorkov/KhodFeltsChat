package schemas

type NewPasswordInput struct {
	// New password of the user.
	// required: true
	// nullable: false
	// example: SomePa$$wordWithDifferentRegisterAndNubmersAndSpecialCharacters228
	// in: body
	NewPassword string `json:"newPassword"`
}

// ForgetPasswordInput
// swagger:parameters ForgetPassword
type ForgetPasswordInput struct {
	// Information about User to forget old password
	// required: true
	// nullable: false
	// in: body
	Body NewPasswordInput

	// Token to define user for forgetting old password
	// required: true
	// nullable: false
	// in: path
	// swagger:strfmt byte
	ForgetPasswordToken string `json:"forgetPasswordToken"`
}

// ChangePasswordInput
// swagger:parameters ChangePassword
type ChangePasswordInput struct {
	// Information about User to change password
	// required: true
	// nullable: false
	// in: body
	Body struct {
		NewPasswordInput

		// Old password of the user.
		// required: true
		// nullable: false
		// example: SomeOldPa$$wordWithDifferentRegisterAndNubmersAndSpecialCharacters228
		OldPassword string `json:"oldPassword"`
	}
}

type EmailInput struct {
	// Email of the user.
	// required: true
	// nullable: false
	// format: email
	// example: alexqwerty@yandex.ru
	// in: body
	Email string `json:"email"`
}

// SendVerifyEmailInput
// swagger:parameters SendVerifyEmailMessage
type SendVerifyEmailInput struct {
	// Information for sending message to User
	// required: true
	// nullable: false
	// in: body
	Body EmailInput
}

// SendForgetPasswordInput
// swagger:parameters SendForgetPasswordMessage
type SendForgetPasswordInput struct {
	// Information for sending message to User
	// required: true
	// nullable: false
	// in: body
	Body EmailInput
}

type UsernameInput struct {
	// Unique username of the user.
	// required: true
	// nullable: false
	// minLength: 5
	// maxLength: 70
	// example: D3M0S
	// in: body
	Username string `json:"username,omitempty"`
}

type PasswordInput struct {
	// Password of the user.
	// required: true
	// nullable: false
	// example: SomePa$$wordWithDifferentRegisterAndNubmersAndSpecialCharacters228
	// in: body
	Password string `json:"password"`
}

// RegisterInput
// swagger:parameters RegisterUser
type RegisterInput struct {
	// Information about User for registration
	// required: true
	// nullable: false
	// in: body
	Body struct {
		EmailInput
		UsernameInput
		PasswordInput
	}
}

// LoginInput
// swagger:parameters Login
type LoginInput struct {
	// Information about User for login
	// required: true
	// nullable: false
	// in: body
	Body struct {
		EmailInput
		PasswordInput
	}
}

// RefreshTokenInput
// swagger:parameters RefreshTokens
type RefreshTokenInput struct {
	// Refresh token to refresh accessToken and refreshToken of user. SHOULD BE PROVIDED IN COOKIE.
	// required: true
	// nullable: false
	// swagger:strfmt byte
	// in: header
	RefreshToken string `json:"refreshToken"`
}

// VerifyEmailInput
// swagger:parameters VerifyEmail
type VerifyEmailInput struct {
	// Token to define user for email verification
	// required: true
	// nullable: false
	// swagger:strfmt byte
	// in: path
	VerifyEmailToken string `json:"verifyEmailToken"`
}
