package schemas

import "time"

// Message represents a message record, which was sent to the chat.
// swagger:model
type Message struct {
	// Unique identifier of the message.
	// required: true
	// nullable: false
	// minimum: 1
	ID uint64 `json:"id"`

	// Unique identifier of chat, in which message was sent.
	// required: true
	// nullable: false
	// minimum: 1
	ChatID uint64 `json:"chatId"`

	// User, who sent the message to the chat.
	// required: true
	// nullable: false
	// example: {"id": 1, "username": "D3M0S"}
	Sender Sender `json:"sender"`

	// Text of the message.
	// required: true
	// nullable: false
	// minLength: 1
	// example: Anything
	Text string `json:"text"`

	// Represents datetime when message was sent.
	// required: true
	// nullable: false
	// format: date-time
	CreatedAt time.Time `json:"createdAt"`

	// Represents datetime when message was updated.
	// required: true
	// nullable: false
	// format: date-time
	UpdatedAt time.Time `json:"updatedAt"`

	// Whether message was read or not for User.
	// required: true
	// nullable: false
	// example: true
	IsRead bool `json:"isRead"`

	// Message that this message is a reply to.
	// required: false
	// nullable: true
	ReplyToMessage *ReplyMessage `json:"replyToMessage,omitempty"`
}

// ReplyMessage represents an abbreviated message that was replied to.
// swagger:model
type ReplyMessage struct {
	// Unique identifier of the original message.
	// required: true
	// nullable: false
	// minimum: 1
	ID uint64 `json:"id"`

	// Sender of the original message.
	// required: true
	// nullable: false
	Sender Sender `json:"sender"`

	// Text of the original message.
	// required: true
	// nullable: false
	Text string `json:"text"`

	// When the original message was sent.
	// required: true
	// nullable: false
	// format: date-time
	CreatedAt time.Time `json:"createdAt"`
}

// Sender represents a user record, who sent message to the chat.
// swagger:model
type Sender struct {
	// Unique identifier of the user.
	// required: true
	// nullable: false
	// minimum: 1
	ID uint64 `json:"id"`

	// Unique username of the user.
	// required: true
	// nullable: false
	// minLength: 5
	// maxLength: 70
	// example: D3M0S
	Username string `json:"username"`
}

// GetChatMessagesInput
// swagger:parameters GetChatMessages
type GetChatMessagesInput struct {
	Pagination
}
