package chats

import (
	"context"
	"fmt"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

type UseCases struct {
	chatsService interfaces.ChatsService
	usersService interfaces.UsersService
}

func New(
	chatsService interfaces.ChatsService,
	usersService interfaces.UsersService,
) *UseCases {
	return &UseCases{
		chatsService: chatsService,
		usersService: usersService,
	}
}

func (u *UseCases) GetChatMembers(
	ctx context.Context,
	chatID, userID uint64,
) ([]domains.User, error) {
	return u.chatsService.GetChatMembers(ctx, chatID, userID)
}

func (u *UseCases) GetUserChats(
	ctx context.Context,
	userID uint64,
	pagination *domains.Pagination,
) ([]domains.Chat, error) {
	if _, err := u.usersService.GetUserByID(ctx, userID); err != nil {
		return nil, err
	}

	return u.chatsService.GetUserChats(ctx, userID, pagination)
}

func (u *UseCases) CreateChat(ctx context.Context, chat domains.Chat) (*domains.Chat, error) {
	if !chat.IsValid() {
		return nil, fmt.Errorf(
			"%w: invalid chat type or members count",
			customerrors.ErrInvalidChat,
		)
	}

	// Проверяем существование пользователей:
	for _, member := range chat.Members {
		if _, err := u.usersService.GetUserByID(ctx, member.ID); err != nil {
			return nil, err
		}
	}

	if chat.Type == domains.ChatTypePrivate {
		exists, err := u.chatsService.PrivateChatExists(ctx, chat.Members)
		if err != nil {
			return nil, err
		}

		if exists {
			return nil, customerrors.ErrChatAlreadyExists
		}
	}

	return u.chatsService.CreateChat(ctx, chat)
}
