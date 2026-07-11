package domains

type WSEventType string

const (
	WSEventNewMessage      WSEventType = "new_message"
	WSEventMessageDeleted  WSEventType = "message_deleted"
	WSEventMessageEdited   WSEventType = "message_edited"
	WSEventReactionAdded   WSEventType = "reaction_added"
	WSEventReactionRemoved WSEventType = "reaction_removed"
)

type WSEvent struct {
	Type    WSEventType `json:"type"`
	Payload any         `json:"payload"`
}

type MessageDeletedPayload struct {
	MessageID uint64 `json:"messageId"`
	ChatID    uint64 `json:"chatId"`
}

type MessageEditedPayload struct {
	MessageID uint64 `json:"messageId"`
	ChatID    uint64 `json:"chatId"`
}

type ReactionAddedPayload struct {
	MessageID  uint64 `json:"messageId"`
	ChatID     uint64 `json:"chatId"`
	UserID     uint64 `json:"userId"`
	ReactionID uint64 `json:"reactionId"`
	Emoji      string `json:"emoji"`
}

type ReactionRemovedPayload struct {
	MessageID  uint64 `json:"messageId"`
	ChatID     uint64 `json:"chatId"`
	UserID     uint64 `json:"userId"`
	ReactionID uint64 `json:"reactionId"`
}
