package errors

import "errors"

var (
	ErrWebPushSubscriptionNotFound = errors.New("web-push subscription not found")
	ErrWebPushSubscriptionExpired  = errors.New("web-push subscription expired")
)
