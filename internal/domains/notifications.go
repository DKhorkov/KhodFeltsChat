package domains

type VerifyEmailNotificationDTO struct {
	UserID uint64 `json:"userId"`
}

type ForgetPasswordNotificationDTO struct {
	UserID uint64 `json:"userId"`
}
