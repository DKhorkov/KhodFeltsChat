package mappers

import (
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
		Members:     MapUsers(chat.Members),
		Messages:    MapMessages(chat.Messages),
	}
}

func MapChats(chats []domains.Chat) []schemas.Chat {
	result := make([]schemas.Chat, len(chats))
	for i, chat := range chats {
		result[i] = MapChat(chat)
	}

	return result
}
