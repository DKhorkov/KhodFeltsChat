package mappers

import (
	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	"github.com/DKhorkov/kfc/internal/domains"
)

func MapMessage(message domains.Message) schemas.Message {
	return schemas.Message{
		ID:     message.ID,
		ChatID: message.ChatID,
		Sender: schemas.Sender{
			ID:       message.Sender.ID,
			Username: message.Sender.Username,
		},
		Text:      message.Text,
		CreatedAt: message.CreatedAt,
		UpdatedAt: message.UpdatedAt,
		IsRead:    message.IsRead,
	}
}

func MapMessages(messages []domains.Message) []schemas.Message {
	result := make([]schemas.Message, len(messages))
	for i := range messages {
		result[i] = MapMessage(messages[i]) // gocritic - чтобы не копировать каждое сообщение
	}

	return result
}
