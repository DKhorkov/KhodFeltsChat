package usecases

import (
	"context"
	"slices"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

type MessagesUseCases struct {
	messagesService interfaces.MessagesService
	usersService    interfaces.UsersService
	chatsService    interfaces.ChatsService
}

func NewMessagesUseCases(
	messagesService interfaces.MessagesService,
	chatsService interfaces.ChatsService,
	usersService interfaces.UsersService,
) *MessagesUseCases {
	return &MessagesUseCases{
		messagesService: messagesService,
		chatsService:    chatsService,
		usersService:    usersService,
	}
}

func (u *MessagesUseCases) SaveMessage(
	ctx context.Context,
	message domains.Message,
) (*domains.Message, error) {
	return u.messagesService.SaveMessage(ctx, message)
}

func (u *MessagesUseCases) GetChatMessages(
	ctx context.Context,
	userID uint64,
	chatID uint64,
	pagination *domains.Pagination,
) ([]domains.Message, error) {
	if _, err := u.usersService.GetUserByID(ctx, userID); err != nil {
		return nil, err
	}

	chatMembers, err := u.chatsService.GetChatMembers(ctx, chatID)
	if err != nil {
		return nil, err
	}

	if !slices.ContainsFunc(
		chatMembers,
		func(member domains.User) bool {
			return member.ID == userID
		},
	) {
		return nil, customerrors.ErrUserIsNotChatMember
	}

	return u.messagesService.GetChatMessages(ctx, userID, chatID, pagination)
}
