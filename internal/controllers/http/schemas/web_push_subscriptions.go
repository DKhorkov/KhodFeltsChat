package schemas

// WebPushSubscriptionKeys represents the browser push subscription keys.
// swagger:model
type WebPushSubscriptionKeys struct {
	// Encryption key (P-256 ECDH)
	// required: true
	EncryptionKey string `json:"encryptionKey"`
	// Auth secret
	// required: true
	Auth string `json:"auth"`
}

// CreateWebPushSubscriptionRequest body
// swagger:parameters CreateWebPushSubscription
type CreateWebPushSubscriptionRequest struct {
	// Push subscription data
	// required: true
	// in: body
	Body struct {
		// Push endpoint URL
		// required: true
		Endpoint string `json:"endpoint"`
		// Push subscription keys
		// required: true
		Keys WebPushSubscriptionKeys `json:"keys"`
	}
}

// CreateWebPushSubscriptionResponse represents the response for a created push subscription.
// swagger:model
type CreateWebPushSubscriptionResponse struct {
	// ID of the created subscription
	ID uint64 `json:"id"`
}

// VAPIDKeyResponse represents the VAPID public key response.
// swagger:model
type VAPIDKeyResponse struct {
	// VAPID public key
	PublicKey string `json:"publicKey"`
}
