package schemas

import "time"

type Chat struct {
	ID          uint64    `json:"id"`
	Title       *string   `json:"title,omitempty"`
	Description *string   `json:"description,omitempty"`
	Type        string    `json:"type"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	IsRead      bool      `json:"isRead"`
	Members     []User    `json:"members,omitempty"`
	Messages    []Message `json:"messages,omitempty"`
}
