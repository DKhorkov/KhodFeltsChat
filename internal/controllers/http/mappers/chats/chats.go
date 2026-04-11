package chats

import (
	"github.com/DKhorkov/kfc/internal/controllers/http/mappers/messages"
	"github.com/DKhorkov/kfc/internal/controllers/http/mappers/users"
	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	"github.com/DKhorkov/kfc/internal/domains"
)

func MapChat(chat domains.Chat) schemas.Chat {
	return schemas.Chat{
		ID:          chat.ID,
		Title:       chat.Title,
		Description: chat.Description,
		Type:        string(chat.Type),
		CreatedAt:   chat.CreatedAt,
		UpdatedAt:   chat.UpdatedAt,
		IsRead:      chat.IsRead,
		Members:     users.MapUsers(chat.Members),
		Messages:    messages.MapMessages(chat.Messages),
	}
}

func MapChats(chats []domains.Chat) []schemas.Chat {
	result := make([]schemas.Chat, len(chats))
	for i := range chats {
		result[i] = MapChat(chats[i]) // gocritic - чтобы не копировать каждое сообщение
	}

	return result
}
