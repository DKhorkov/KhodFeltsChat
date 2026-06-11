package schemas

import "time"

// Chat represents a chat record, where users can send a message.
// swagger:model
type Chat struct {
	// Unique identifier of the chat.
	// required: true
	// nullable: false
	// minimum: 1
	ID uint64 `json:"id"`

	// Title of the chat.
	// required: false
	// nullable: true
	// minLength: 1
	// maxLength: 100
	// example: Party
	Title *string `json:"title,omitempty"`

	// Description of the chat.
	// required: false
	// nullable: true
	// minLength: 1
	// example: Long description of a party
	Description *string `json:"description,omitempty"`

	// Type of the chat.
	// required: true
	// nullable: false
	// minLength: 1
	// maxLength: 40
	// enum: private,group
	// example: private
	Type string `json:"type"`

	// Represents datetime when chat was created.
	// required: true
	// nullable: false
	// format: date-time
	CreatedAt time.Time `json:"createdAt"`

	// Represents datetime when chat was updated.
	// required: true
	// nullable: false
	// format: date-time
	UpdatedAt time.Time `json:"updatedAt"`

	// Whether chat was read or not.
	// required: true
	// nullable: false
	// example: true
	IsRead bool `json:"isRead"`

	// Number of unread non-deleted messages in the chat for the current user.
	// required: true
	// nullable: false
	// minimum: 0
	// example: 3
	UnreadCount uint64 `json:"unreadCount"`

	// Members of the chat.
	// required: false
	// nullable: true
	Members []User `json:"members,omitempty"`

	// Messages which were sent to the chat.
	// required: false
	// nullable: true
	Messages []Message `json:"messages,omitempty"`
}

// CreateChatInput
// swagger:parameters CreateChat
type CreateChatInput struct {
	// Information about Chat for creation
	// required: true
	// nullable: false
	// in: body
	Body struct {
		// Type of the chat.
		// required: true
		// nullable: false
		// minLength: 1
		// maxLength: 40
		// enum: private,group
		// example: private
		Type string `json:"type"`

		// Members of the chat. Current User should NOT be provided directly!
		// required: true
		// nullable: false
		Members []MemberInput `json:"members"`
	}
}

// MemberInput represents an input of members unique identifier for creating chat.
// swagger:model
type MemberInput struct {
	// Unique identifier of the user.
	// required: true
	// nullable: false
	// minimum: 1
	ID uint64 `json:"id"`
}

// GetUserChatsInput
// swagger:parameters GetUserChats
type GetUserChatsInput struct {
	Pagination
}
