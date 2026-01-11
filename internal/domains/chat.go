package domains

import (
	"slices"
	"time"
)

type ChatType string

const (
	ChatTypePrivate ChatType = "private"
	ChatTypeGroup   ChatType = "group"

	minMembersCount = 1 // группа на одного
)

var chatTypes = []ChatType{
	ChatTypePrivate,
	ChatTypeGroup,
}

type Chat struct {
	ID          uint64    `json:"id"`
	Title       *string   `json:"title,omitempty"`
	Description *string   `json:"description,omitempty"`
	Type        ChatType  `json:"type"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	IsRead      bool      `json:"isRead"`
	Members     []User    `json:"members,omitempty"`
	Messages    []Message `json:"messages,omitempty"`
}

func (c *Chat) IsValid() bool {
	if !slices.Contains(chatTypes, c.Type) {
		return false
	}

	if len(c.Members) < minMembersCount {
		return false
	}

	return true
}
