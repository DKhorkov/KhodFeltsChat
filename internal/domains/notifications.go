package domains

type VerifyEmailNotificationDTO struct {
	UserID uint64 `json:"userId"`
}

type ForgetPasswordNotificationDTO struct {
	UserID uint64 `json:"userId"`
}

type WebPushNotificationDTO struct {
	UserID    uint64 `json:"userId"`
	MessageID uint64 `json:"messageId"`
}
