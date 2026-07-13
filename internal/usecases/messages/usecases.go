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
	messagesService  interfaces.MessagesService
	usersService     interfaces.UsersService
	chatsService     interfaces.ChatsService
	reactionsService interfaces.ReactionsService
}

func New(
	messagesService interfaces.MessagesService,
	chatsService interfaces.ChatsService,
	usersService interfaces.UsersService,
	reactionsService interfaces.ReactionsService,
) *UseCases {
	return &UseCases{
		messagesService:  messagesService,
		chatsService:     chatsService,
		usersService:     usersService,
		reactionsService: reactionsService,
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

	chatMembers, err := u.chatsService.GetChatMembers(ctx, chatID, userID)
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

	msgs, err := u.messagesService.GetChatMessages(ctx, userID, chatID, pagination)
	if err != nil {
		return nil, err
	}

	return u.attachReactions(ctx, msgs)
}

func (u *UseCases) GetMessageByID(
	ctx context.Context,
	userID uint64,
	messageID uint64,
) (*domains.Message, error) {
	msg, err := u.messagesService.GetMessageByID(ctx, userID, messageID)
	if err != nil {
		return nil, err
	}

	enriched, err := u.attachReactions(ctx, []domains.Message{*msg})
	if err != nil {
		return nil, err
	}

	return &enriched[0], nil
}

func (u *UseCases) GetUserUnreadCount(
	ctx context.Context,
	userID uint64,
) (uint64, error) {
	return u.messagesService.GetUserUnreadCount(ctx, userID)
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

func (u *UseCases) UpdateMessage(
	ctx context.Context,
	dto domains.UpdateMessageDTO,
) (*domains.Message, error) {
	message, err := u.messagesService.GetMessageByID(ctx, dto.UserID, dto.MessageID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", customerrors.ErrMessageNotFound, err)
	}

	if message.Sender.ID != dto.UserID {
		return nil, customerrors.ErrNotMessageAuthor
	}

	return u.messagesService.UpdateMessage(ctx, dto)
}

// attachReactions обогащает сообщения реакциями пачкой.
func (u *UseCases) attachReactions(
	ctx context.Context,
	msgs []domains.Message,
) ([]domains.Message, error) {
	if len(msgs) == 0 {
		return msgs, nil
	}

	ids := make([]uint64, 0, len(msgs))
	for i := range msgs {
		ids = append(ids, msgs[i].ID)
	}

	byMsg, err := u.reactionsService.ListReactionsForMessages(ctx, ids)
	if err != nil {
		return nil, err
	}

	for i := range msgs {
		msgs[i].Reactions = byMsg[msgs[i].ID]
	}

	return msgs, nil
}
