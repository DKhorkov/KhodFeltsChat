package messages

import (
	reactionsmapper "github.com/DKhorkov/kfc/internal/controllers/http/mappers/reactions"
	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	"github.com/DKhorkov/kfc/internal/domains"
)

func MapMessage(message domains.Message) schemas.Message {
	mapped := schemas.Message{
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

	if message.ReplyToMessage != nil {
		mapped.ReplyToMessage = &schemas.ReplyMessage{
			ID: message.ReplyToMessage.ID,
			Sender: schemas.Sender{
				ID:       message.ReplyToMessage.Sender.ID,
				Username: message.ReplyToMessage.Sender.Username,
			},
			Text:      message.ReplyToMessage.Text,
			CreatedAt: message.ReplyToMessage.CreatedAt,
		}
	}

	mapped.Reactions = reactionsmapper.MapMessageReactions(message.Reactions)

	return mapped
}

func MapMessages(messages []domains.Message) []schemas.Message {
	result := make([]schemas.Message, len(messages))
	for i := range messages {
		result[i] = MapMessage(messages[i]) // gocritic - чтобы не копировать каждое сообщение
	}

	return result
}
