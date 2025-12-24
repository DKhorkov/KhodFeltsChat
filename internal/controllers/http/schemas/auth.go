package schemas

type ForgetPasswordDTO struct {
	NewPassword string `json:"newPassword"`
}

type ChangePasswordDTO struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

type SendVerifyEmailDTO struct {
	Email string `json:"email"`
}

type SendForgetPasswordDTO struct {
	Email string `json:"email"`
}
