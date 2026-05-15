package domains

import "time"

type WebPushSubscription struct {
	ID            uint64    `json:"id"`
	UserID        uint64    `json:"userId"`
	Endpoint      string    `json:"endpoint"`
	EncryptionKey string    `json:"encryptionKey"`
	Auth          string    `json:"auth"`
	CreatedAt     time.Time `json:"createdAt"`
}
