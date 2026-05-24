package messages

import (
	"context"
	"fmt"
	"slices"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

type UseCases struct {
	messagesService interfaces.MessagesService
	usersService    interfaces.UsersService
	chatsService    interfaces.ChatsService
}

func New(
	messagesService interfaces.MessagesService,
	chatsService interfaces.ChatsService,
	usersService interfaces.UsersService,
) *UseCases {
	return &UseCases{
		messagesService: messagesService,
		chatsService:    chatsService,
		usersService:    usersService,
	}
}

func (u *UseCases) SaveMessage(
	ctx context.Context,
	message domains.Message,
) (*domains.Message, error) {
	return u.messagesService.SaveMessage(ctx, message)
}

func (u *UseCases) GetChatMessages(
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

func (u *UseCases) GetMessageByID(
	ctx context.Context,
	userID uint64,
	messageID uint64,
) (*domains.Message, error) {
	return u.messagesService.GetMessageByID(ctx, userID, messageID)
}

func (u *UseCases) DeleteMessage(
	ctx context.Context,
	dto domains.DeleteMessageDTO,
) error {
	if dto.ForAll {
		message, err := u.messagesService.GetMessageByID(ctx, dto.UserID, dto.MessageID)
		if err != nil {
			return fmt.Errorf("%w: %w", customerrors.ErrMessageNotFound, err)
		}

		if message.Sender.ID != dto.UserID {
			return customerrors.ErrNotMessageAuthor
		}
	}

	return u.messagesService.DeleteMessage(ctx, dto)
}
