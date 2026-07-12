package reactions

import (
	"context"
	"slices"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

type UseCases struct {
	reactionsService interfaces.ReactionsService
	messagesService  interfaces.MessagesService
	chatsService     interfaces.ChatsService
}

func New(
	reactionsService interfaces.ReactionsService,
	messagesService interfaces.MessagesService,
	chatsService interfaces.ChatsService,
) *UseCases {
	return &UseCases{
		reactionsService: reactionsService,
		messagesService:  messagesService,
		chatsService:     chatsService,
	}
}

func (u *UseCases) ListReactions(ctx context.Context) ([]domains.Reaction, error) {
	return u.reactionsService.ListReactions(ctx)
}

func (u *UseCases) AddReaction(
	ctx context.Context,
	dto domains.MessageReactionDTO,
) (*domains.Reaction, error) {
	message, err := u.messagesService.GetMessageByID(ctx, dto.UserID, dto.MessageID)
	if err != nil {
		return nil, err
	}

	if err = u.ensureUserIsChatMember(ctx, message.ChatID, dto.UserID); err != nil {
		return nil, err
	}

	reaction, err := u.reactionsService.GetReactionByID(ctx, dto.ReactionID)
	if err != nil {
		return nil, err
	}

	if err = u.reactionsService.AddMessageReaction(ctx, dto); err != nil {
		return nil, err
	}

	return reaction, nil
}

func (u *UseCases) RemoveReaction(
	ctx context.Context,
	dto domains.MessageReactionDTO,
) error {
	message, err := u.messagesService.GetMessageByID(ctx, dto.UserID, dto.MessageID)
	if err != nil {
		return err
	}

	if err = u.ensureUserIsChatMember(ctx, message.ChatID, dto.UserID); err != nil {
		return err
	}

	return u.reactionsService.RemoveMessageReaction(ctx, dto)
}

func (u *UseCases) ensureUserIsChatMember(
	ctx context.Context,
	chatID, userID uint64,
) error {
	members, err := u.chatsService.GetChatMembers(ctx, chatID, userID)
	if err != nil {
		return err
	}

	if !slices.ContainsFunc(
		members,
		func(m domains.User) bool { return m.ID == userID },
	) {
		return customerrors.ErrUserIsNotChatMember
	}

	return nil
}
