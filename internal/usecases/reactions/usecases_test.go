package reactions_test

import (
	"context"
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

func TestUseCases_AddReaction_HappyPath_ReturnsReaction(t *testing.T) {
	t.Parallel()

	uc, d := newUseCase(t)
	ctx := context.Background()
	dto := domains.MessageReactionDTO{MessageID: 10, ReactionID: 1, UserID: 7}
	expected := &domains.Reaction{ID: 1, Emoji: "👍"}

	d.messages.EXPECT().
		GetMessageByID(ctx, uint64(7), uint64(10)).
		Return(&domains.Message{ID: 10, ChatID: 42}, nil)
	d.chats.EXPECT().
		GetChatMembers(ctx, uint64(42), uint64(7)).
		Return([]domains.User{{ID: 7}, {ID: 8}}, nil)
	d.reactions.EXPECT().
		GetReactionByID(ctx, uint64(1)).
		Return(expected, nil)
	d.reactions.EXPECT().
		AddMessageReaction(ctx, dto).
		Return(nil)

	got, err := uc.AddReaction(ctx, dto)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestUseCases_AddReaction_MessageNotFound(t *testing.T) {
	t.Parallel()

	uc, d := newUseCase(t)
	dto := domains.MessageReactionDTO{MessageID: 10, ReactionID: 1, UserID: 7}

	d.messages.EXPECT().
		GetMessageByID(gomock.Any(), uint64(7), uint64(10)).
		Return(nil, customerrors.ErrMessageNotFound)

	got, err := uc.AddReaction(context.Background(), dto)
	assert.ErrorIs(t, err, customerrors.ErrMessageNotFound)
	assert.Nil(t, got)
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

	got, err := uc.AddReaction(context.Background(), dto)
	assert.ErrorIs(t, err, customerrors.ErrUserIsNotChatMember)
	assert.Nil(t, got)
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

	got, err := uc.AddReaction(context.Background(), dto)
	assert.ErrorIs(t, err, customerrors.ErrReactionNotFound)
	assert.Nil(t, got)
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

	got, err := uc.AddReaction(context.Background(), dto)
	assert.ErrorIs(t, err, customerrors.ErrReactionAlreadyExists)
	assert.Nil(t, got)
}

func TestUseCases_RemoveReaction_HappyPath(t *testing.T) {
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

	assert.NoError(t, uc.RemoveReaction(ctx, dto))
}

func TestUseCases_RemoveReaction_NotSet(t *testing.T) {
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

	err := uc.RemoveReaction(context.Background(), dto)
	assert.ErrorIs(t, err, customerrors.ErrReactionNotSet)
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

	err := uc.RemoveReaction(context.Background(), dto)
	assert.ErrorIs(t, err, customerrors.ErrUserIsNotChatMember)
}
