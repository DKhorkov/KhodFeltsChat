package domains

import "time"

type ThemeType int

const (
	ThemeLight ThemeType = iota
	ThemeDark
)

type Settings struct {
	ID        uint64    `json:"id"`
	UserID    uint64    `json:"userID"`
	Theme     ThemeType `json:"theme"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
