package domains

import "encoding/json"

// EmailNotificationType represents a type of email notification.
type EmailNotificationType string

const (
	EmailTypeVerifyEmail    EmailNotificationType = "verify_email"
	EmailTypeForgetPassword EmailNotificationType = "forget_password"
	EmailTypeNewMessage     EmailNotificationType = "new_message"
)

type EmailNotificationDTO struct {
	Type    EmailNotificationType `json:"type"`
	UserID  uint64                `json:"userId"`
	Payload json.RawMessage       `json:"payload"`
}

// WebPushNotificationType represents a type of web push notification.
type WebPushNotificationType string

const (
	WebPushTypeNewMessage WebPushNotificationType = "new_message"
)

type WebPushNotificationDTO struct {
	Type    WebPushNotificationType `json:"type"`
	UserID  uint64                  `json:"userId"`
	Payload json.RawMessage         `json:"payload"`
}

// NewMessagePayload contains data for new message notifications.
type NewMessagePayload struct {
	MessageID uint64 `json:"messageId"`
}
