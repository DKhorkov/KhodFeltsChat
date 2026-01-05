package usecases

import (
	"github.com/DKhorkov/kfc/internal/domains"
)

type ChatsUseCases struct{}

func NewChatsUseCases() *ChatsUseCases {
	return &ChatsUseCases{}
}

func (u *ChatsUseCases) GetChatMembers(chatID string) ([]domains.User, error) {
	return []domains.User{
		{
			ID:       8,
			Username: "test1",
		},
		{
			ID:       9,
			Username: "test2",
		},
		{
			ID:       11,
			Username: "test3",
		},
	}, nil
}
