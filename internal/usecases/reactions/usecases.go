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

// AddReaction возвращает chatID и emoji, чтобы HTTP-handler мог после
// успешного вызова опубликовать WS-событие. Broadcast делает handler,
// не usecase — так у usecase нет обратной зависимости на *ws.Handler.
func (u *UseCases) AddReaction(
	ctx context.Context,
	dto domains.MessageReactionDTO,
) (uint64, string, error) {
	message, err := u.messagesService.GetMessageByID(ctx, dto.UserID, dto.MessageID)
	if err != nil {
		return 0, "", err
	}

	if err = u.ensureUserIsChatMember(ctx, message.ChatID, dto.UserID); err != nil {
		return 0, "", err
	}

	reaction, err := u.reactionsService.GetReactionByID(ctx, dto.ReactionID)
	if err != nil {
		return 0, "", err
	}

	if err = u.reactionsService.AddMessageReaction(ctx, dto); err != nil {
		return 0, "", err
	}

	return message.ChatID, reaction.Emoji, nil
}

// RemoveReaction возвращает chatID для последующего broadcast'а. При
// отсутствии реакции — customerrors.ErrReactionNotSet; handler трактует
// это как идемпотентный успех (200 без publish'а).
func (u *UseCases) RemoveReaction(
	ctx context.Context,
	dto domains.MessageReactionDTO,
) (uint64, error) {
	message, err := u.messagesService.GetMessageByID(ctx, dto.UserID, dto.MessageID)
	if err != nil {
		return 0, err
	}

	if err = u.ensureUserIsChatMember(ctx, message.ChatID, dto.UserID); err != nil {
		return 0, err
	}

	if err = u.reactionsService.RemoveMessageReaction(ctx, dto); err != nil {
		return message.ChatID, err
	}

	return message.ChatID, nil
}

func (u *UseCases) AttachReactions(
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

func (u *UseCases) AttachReaction(
	ctx context.Context,
	msg *domains.Message,
) (*domains.Message, error) {
	if msg == nil {
		return nil, nil
	}

	byMsg, err := u.reactionsService.ListReactionsForMessages(ctx, []uint64{msg.ID})
	if err != nil {
		return nil, err
	}

	msg.Reactions = byMsg[msg.ID]

	return msg, nil
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
