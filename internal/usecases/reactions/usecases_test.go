package reactions_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	reactionsusecases "github.com/DKhorkov/kfc/internal/usecases/reactions"
	mockservices "github.com/DKhorkov/kfc/mocks/services"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type usecaseDeps struct {
	reactions *mockservices.MockReactionsService
	messages  *mockservices.MockMessagesService
	chats     *mockservices.MockChatsService
}

func newUseCase(t *testing.T) (*reactionsusecases.UseCases, usecaseDeps) {
	t.Helper()

	ctrl := gomock.NewController(t)
	d := usecaseDeps{
		reactions: mockservices.NewMockReactionsService(ctrl),
		messages:  mockservices.NewMockMessagesService(ctrl),
		chats:     mockservices.NewMockChatsService(ctrl),
	}

	return reactionsusecases.New(d.reactions, d.messages, d.chats), d
}

func TestUseCases_ListReactions(t *testing.T) {
	t.Parallel()

	uc, d := newUseCase(t)
	expected := []domains.Reaction{{ID: 1, Emoji: "👍"}}
	d.reactions.EXPECT().ListReactions(gomock.Any()).Return(expected, nil)

	got, err := uc.ListReactions(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestUseCases_AddReaction_HappyPath_ReturnsChatIDAndEmoji(t *testing.T) {
	t.Parallel()

	uc, d := newUseCase(t)
	ctx := context.Background()
	dto := domains.MessageReactionDTO{MessageID: 10, ReactionID: 1, UserID: 7}

	d.messages.EXPECT().
		GetMessageByID(ctx, uint64(7), uint64(10)).
		Return(&domains.Message{ID: 10, ChatID: 42}, nil)
	d.chats.EXPECT().
		GetChatMembers(ctx, uint64(42), uint64(7)).
		Return([]domains.User{{ID: 7}, {ID: 8}}, nil)
	d.reactions.EXPECT().
		GetReactionByID(ctx, uint64(1)).
		Return(&domains.Reaction{ID: 1, Emoji: "👍"}, nil)
	d.reactions.EXPECT().
		AddMessageReaction(ctx, dto).
		Return(nil)

	chatID, emoji, err := uc.AddReaction(ctx, dto)
	assert.NoError(t, err)
	assert.Equal(t, uint64(42), chatID)
	assert.Equal(t, "👍", emoji)
}

func TestUseCases_AddReaction_MessageNotFound(t *testing.T) {
	t.Parallel()

	uc, d := newUseCase(t)
	dto := domains.MessageReactionDTO{MessageID: 10, ReactionID: 1, UserID: 7}

	d.messages.EXPECT().
		GetMessageByID(gomock.Any(), uint64(7), uint64(10)).
		Return(nil, customerrors.ErrMessageNotFound)

	chatID, emoji, err := uc.AddReaction(context.Background(), dto)
	assert.ErrorIs(t, err, customerrors.ErrMessageNotFound)
	assert.Equal(t, uint64(0), chatID)
	assert.Equal(t, "", emoji)
}

func TestUseCases_AddReaction_UserNotChatMember(t *testing.T) {
	t.Parallel()

	uc, d := newUseCase(t)
	dto := domains.MessageReactionDTO{MessageID: 10, ReactionID: 1, UserID: 7}

	d.messages.EXPECT().
		GetMessageByID(gomock.Any(), uint64(7), uint64(10)).
		Return(&domains.Message{ID: 10, ChatID: 42}, nil)
	d.chats.EXPECT().
		GetChatMembers(gomock.Any(), uint64(42), uint64(7)).
		Return([]domains.User{{ID: 99}}, nil)

	_, _, err := uc.AddReaction(context.Background(), dto)
	assert.ErrorIs(t, err, customerrors.ErrUserIsNotChatMember)
}

func TestUseCases_AddReaction_UnknownReaction(t *testing.T) {
	t.Parallel()

	uc, d := newUseCase(t)
	dto := domains.MessageReactionDTO{MessageID: 10, ReactionID: 999, UserID: 7}

	d.messages.EXPECT().
		GetMessageByID(gomock.Any(), uint64(7), uint64(10)).
		Return(&domains.Message{ID: 10, ChatID: 42}, nil)
	d.chats.EXPECT().
		GetChatMembers(gomock.Any(), uint64(42), uint64(7)).
		Return([]domains.User{{ID: 7}}, nil)
	d.reactions.EXPECT().
		GetReactionByID(gomock.Any(), uint64(999)).
		Return(nil, customerrors.ErrReactionNotFound)

	_, _, err := uc.AddReaction(context.Background(), dto)
	assert.ErrorIs(t, err, customerrors.ErrReactionNotFound)
}

func TestUseCases_AddReaction_Duplicate(t *testing.T) {
	t.Parallel()

	uc, d := newUseCase(t)
	dto := domains.MessageReactionDTO{MessageID: 10, ReactionID: 1, UserID: 7}

	d.messages.EXPECT().
		GetMessageByID(gomock.Any(), uint64(7), uint64(10)).
		Return(&domains.Message{ID: 10, ChatID: 42}, nil)
	d.chats.EXPECT().
		GetChatMembers(gomock.Any(), uint64(42), uint64(7)).
		Return([]domains.User{{ID: 7}}, nil)
	d.reactions.EXPECT().
		GetReactionByID(gomock.Any(), uint64(1)).
		Return(&domains.Reaction{ID: 1, Emoji: "👍"}, nil)
	d.reactions.EXPECT().
		AddMessageReaction(gomock.Any(), dto).
		Return(customerrors.ErrReactionAlreadyExists)

	_, _, err := uc.AddReaction(context.Background(), dto)
	assert.ErrorIs(t, err, customerrors.ErrReactionAlreadyExists)
}

func TestUseCases_RemoveReaction_HappyPath_ReturnsChatID(t *testing.T) {
	t.Parallel()

	uc, d := newUseCase(t)
	ctx := context.Background()
	dto := domains.MessageReactionDTO{MessageID: 10, ReactionID: 1, UserID: 7}

	d.messages.EXPECT().
		GetMessageByID(ctx, uint64(7), uint64(10)).
		Return(&domains.Message{ID: 10, ChatID: 42}, nil)
	d.chats.EXPECT().
		GetChatMembers(ctx, uint64(42), uint64(7)).
		Return([]domains.User{{ID: 7}}, nil)
	d.reactions.EXPECT().
		RemoveMessageReaction(ctx, dto).
		Return(nil)

	chatID, err := uc.RemoveReaction(ctx, dto)
	assert.NoError(t, err)
	assert.Equal(t, uint64(42), chatID)
}

func TestUseCases_RemoveReaction_NotSet_ReturnsChatIDAndError(t *testing.T) {
	t.Parallel()

	uc, d := newUseCase(t)
	dto := domains.MessageReactionDTO{MessageID: 10, ReactionID: 1, UserID: 7}

	d.messages.EXPECT().
		GetMessageByID(gomock.Any(), uint64(7), uint64(10)).
		Return(&domains.Message{ID: 10, ChatID: 42}, nil)
	d.chats.EXPECT().
		GetChatMembers(gomock.Any(), uint64(42), uint64(7)).
		Return([]domains.User{{ID: 7}}, nil)
	d.reactions.EXPECT().
		RemoveMessageReaction(gomock.Any(), dto).
		Return(customerrors.ErrReactionNotSet)

	chatID, err := uc.RemoveReaction(context.Background(), dto)
	assert.ErrorIs(t, err, customerrors.ErrReactionNotSet)
	// chatID отдаётся даже при NotSet — на случай, если вызывающий захочет
	// использовать его для чего-то ещё; handler игнорирует.
	assert.Equal(t, uint64(42), chatID)
}

func TestUseCases_RemoveReaction_UserNotChatMember(t *testing.T) {
	t.Parallel()

	uc, d := newUseCase(t)
	dto := domains.MessageReactionDTO{MessageID: 10, ReactionID: 1, UserID: 7}

	d.messages.EXPECT().
		GetMessageByID(gomock.Any(), uint64(7), uint64(10)).
		Return(&domains.Message{ID: 10, ChatID: 42}, nil)
	d.chats.EXPECT().
		GetChatMembers(gomock.Any(), uint64(42), uint64(7)).
		Return([]domains.User{{ID: 99}}, nil)

	_, err := uc.RemoveReaction(context.Background(), dto)
	assert.ErrorIs(t, err, customerrors.ErrUserIsNotChatMember)
}

func TestUseCases_AttachReactions_EmptyInput_NoServiceCall(t *testing.T) {
	t.Parallel()

	uc, _ := newUseCase(t)

	got, err := uc.AttachReactions(context.Background(), nil)
	assert.NoError(t, err)
	assert.Nil(t, got)

	got, err = uc.AttachReactions(context.Background(), []domains.Message{})
	assert.NoError(t, err)
	assert.Empty(t, got)
}

func TestUseCases_AttachReactions_MapsByMessageID(t *testing.T) {
	t.Parallel()

	uc, d := newUseCase(t)
	msgs := []domains.Message{{ID: 10}, {ID: 20}}

	d.reactions.EXPECT().
		ListReactionsForMessages(gomock.Any(), []uint64{10, 20}).
		Return(map[uint64][]domains.MessageReactionSummary{
			10: {{Reaction: domains.Reaction{ID: 1, Emoji: "👍"}, UserIDs: []uint64{1}}},
			20: {{Reaction: domains.Reaction{ID: 2, Emoji: "❤️"}, UserIDs: []uint64{2, 3}}},
		}, nil)

	got, err := uc.AttachReactions(context.Background(), msgs)
	assert.NoError(t, err)
	assert.Len(t, got[0].Reactions, 1)
	assert.Equal(t, "👍", got[0].Reactions[0].Reaction.Emoji)
	assert.Len(t, got[1].Reactions, 1)
	assert.Equal(t, "❤️", got[1].Reactions[0].Reaction.Emoji)
}

func TestUseCases_AttachReactions_ServiceError(t *testing.T) {
	t.Parallel()

	uc, d := newUseCase(t)
	msgs := []domains.Message{{ID: 10}}
	boom := errors.New("boom")

	d.reactions.EXPECT().
		ListReactionsForMessages(gomock.Any(), []uint64{10}).
		Return(nil, boom)

	got, err := uc.AttachReactions(context.Background(), msgs)
	assert.ErrorIs(t, err, boom)
	assert.Nil(t, got)
}

func TestUseCases_AttachReaction_Nil_ReturnsNil(t *testing.T) {
	t.Parallel()

	uc, _ := newUseCase(t)

	got, err := uc.AttachReaction(context.Background(), nil)
	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestUseCases_AttachReaction_SetsReactions(t *testing.T) {
	t.Parallel()

	uc, d := newUseCase(t)
	msg := &domains.Message{ID: 10}
	summary := []domains.MessageReactionSummary{
		{Reaction: domains.Reaction{ID: 1, Emoji: "👍"}, UserIDs: []uint64{7}},
	}

	d.reactions.EXPECT().
		ListReactionsForMessages(gomock.Any(), []uint64{10}).
		Return(map[uint64][]domains.MessageReactionSummary{10: summary}, nil)

	got, err := uc.AttachReaction(context.Background(), msg)
	assert.NoError(t, err)
	assert.Equal(t, summary, got.Reactions)
}
