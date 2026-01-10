package usecases

import (
	"context"
	"fmt"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

type ChatsUseCases struct {
	chatsService interfaces.ChatsService
	usersService interfaces.UsersService
}

func NewChatsUseCases(
	chatsService interfaces.ChatsService,
	usersService interfaces.UsersService,
) *ChatsUseCases {
	return &ChatsUseCases{
		chatsService: chatsService,
		usersService: usersService,
	}
}

func (u *ChatsUseCases) GetChatMembers(ctx context.Context, chatID uint64) ([]domains.User, error) {
	return u.chatsService.GetChatMembers(ctx, chatID)
}

func (u *ChatsUseCases) GetUserChats(
	ctx context.Context,
	userID uint64,
	pagination *domains.Pagination,
) ([]domains.Chat, error) {
	if _, err := u.usersService.GetUserByID(ctx, userID); err != nil {
		return nil, err
	}

	return u.chatsService.GetUserChats(ctx, userID, pagination)
}

func (u *ChatsUseCases) CreateChat(ctx context.Context, chat domains.Chat) (*domains.Chat, error) {
	if !chat.IsValid() {
		return nil, fmt.Errorf(
			"%w: invalid chat type or members count",
			customerrors.ErrInvalidChat,
		)
	}

	return u.chatsService.CreateChat(ctx, chat)
}
