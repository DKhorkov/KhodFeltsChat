package domains

type WSEventType string

const (
	WSEventNewMessage     WSEventType = "new_message"
	WSEventMessageDeleted WSEventType = "message_deleted"
	WSEventMessageEdited  WSEventType = "message_edited"
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
