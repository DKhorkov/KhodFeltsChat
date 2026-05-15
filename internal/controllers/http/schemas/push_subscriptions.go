package schemas

// PushSubscriptionKeys represents the browser push subscription keys.
// swagger:model
type PushSubscriptionKeys struct {
	// Encryption key (P-256 ECDH)
	// required: true
	EncryptionKey string `json:"encryptionKey"`
	// Auth secret
	// required: true
	Auth string `json:"auth"`
}

// CreatePushSubscriptionRequest body
// swagger:parameters CreatePushSubscription
type CreatePushSubscriptionRequest struct {
	// Push subscription data
	// required: true
	// in: body
	Body struct {
		// Push endpoint URL
		// required: true
		Endpoint string `json:"endpoint"`
		// Push subscription keys
		// required: true
		Keys PushSubscriptionKeys `json:"keys"`
	}
}

// CreatePushSubscriptionResponse represents the response for a created push subscription.
// swagger:model
type CreatePushSubscriptionResponse struct {
	// ID of the created subscription
	ID uint64 `json:"id"`
}

// VAPIDKeyResponse represents the VAPID public key response.
// swagger:model
type VAPIDKeyResponse struct {
	// VAPID public key
	PublicKey string `json:"publicKey"`
}
