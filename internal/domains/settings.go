package domains

import "time"

type ThemeType int

const (
	ThemeLight ThemeType = iota
	ThemeDark
)

type NotificationConsent int

const (
	ConsentNewMessage NotificationConsent = 1 << iota
)

func HasConsent(mask, consent NotificationConsent) bool {
	return mask&consent != 0
}

// Settings fields order must match DB column order for pg.GetEntityColumns scan.
type Settings struct {
	ID              uint64              `json:"id"`
	UserID          uint64              `json:"userId"`
	Theme           ThemeType           `json:"theme"`
	CreatedAt       time.Time           `json:"createdAt"`
	UpdatedAt       time.Time           `json:"updatedAt"`
	EmailConsents   NotificationConsent `json:"emailConsents"`
	WebPushConsents NotificationConsent `json:"webPushConsents"`
}
