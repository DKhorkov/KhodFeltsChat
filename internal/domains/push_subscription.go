package domains

import "time"

type PushSubscription struct {
	ID        uint64    `json:"id"`
	UserID    uint64    `json:"userId"`
	Endpoint  string    `json:"endpoint"`
	P256dh    string    `json:"p256dh"`
	Auth      string    `json:"auth"`
	CreatedAt time.Time `json:"createdAt"`
}
