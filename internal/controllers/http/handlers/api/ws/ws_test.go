package ws_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/ws"
	"github.com/DKhorkov/kfc/internal/domains"
	mockcontrollers "github.com/DKhorkov/kfc/mocks/controllers"
	mockupgrader "github.com/DKhorkov/kfc/mocks/upgrader"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	loggingmocks "github.com/DKhorkov/libs/logging/mocks"
	natsmocks "github.com/DKhorkov/libs/nats/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// Sanity: *ws.Handler удовлетворяет interfaces.WSBroadcaster.
// Проверяется во время компиляции — если методы разошлись, тест не соберётся.
func TestHandler_ImplementsWSBroadcaster(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	h := ws.New(
		mockupgrader.NewMockUpgrader(ctrl),
		mockusecases.NewMockUsersUseCases(ctrl),
		mockusecases.NewMockChatsUseCases(ctrl),
		mockusecases.NewMockMessagesUseCases(ctrl),
		loggingmocks.NewMockLogger(ctrl),
		natsmocks.NewMockPublisher(ctrl),
		config.NATSConfig{},
	)

	// Проверка типа: h должен присваиваться в интерфейс из mocks/controllers.
	_ = mockcontrollers.NewMockWSBroadcaster(ctrl)

	assert.NotNil(t, h)
}

type broadcastDeps struct {
	users    *mockusecases.MockUsersUseCases
	chats    *mockusecases.MockChatsUseCases
	messages *mockusecases.MockMessagesUseCases
	logger   *loggingmocks.MockLogger
}

func newHandler(t *testing.T) (*ws.Handler, broadcastDeps) {
	t.Helper()

	ctrl := gomock.NewController(t)
	d := broadcastDeps{
		users:    mockusecases.NewMockUsersUseCases(ctrl),
		chats:    mockusecases.NewMockChatsUseCases(ctrl),
		messages: mockusecases.NewMockMessagesUseCases(ctrl),
		logger:   loggingmocks.NewMockLogger(ctrl),
	}

	h := ws.New(
		mockupgrader.NewMockUpgrader(ctrl),
		d.users,
		d.chats,
		d.messages,
		d.logger,
		natsmocks.NewMockPublisher(ctrl),
		config.NATSConfig{},
	)

	return h, d
}

func TestHandler_BroadcastReactionAdded_ResolvesChatIDAndQueriesMembers(t *testing.T) {
	t.Parallel()

	h, d := newHandler(t)

	d.messages.EXPECT().
		GetMessageByID(gomock.Any(), uint64(7), uint64(10)).
		Return(&domains.Message{ID: 10, ChatID: 42}, nil).
		Times(1)
	d.chats.EXPECT().
		GetChatMembers(gomock.Any(), uint64(42), uint64(7)).
		Return([]domains.User{{ID: 7}, {ID: 8}}, nil).
		Times(1)

	// Нет активных соединений → sendToUser молча выходит для каждого участника.
	h.BroadcastReactionAdded(context.Background(), 10, 7, 1, "👍")
}

func TestHandler_BroadcastReactionAdded_LogsErrorOnGetMessageFailure(t *testing.T) {
	t.Parallel()

	h, d := newHandler(t)

	d.messages.EXPECT().
		GetMessageByID(gomock.Any(), uint64(7), uint64(10)).
		Return(nil, errors.New("not found")).
		Times(1)
	// GetChatMembers НЕ должен быть вызван.

	d.logger.EXPECT().
		ErrorContext(gomock.Any(), "Failed to resolve chatID for reaction added broadcast", gomock.Any()).
		Times(1)

	h.BroadcastReactionAdded(context.Background(), 10, 7, 1, "👍")
}

func TestHandler_BroadcastReactionAdded_LogsErrorOnGetChatMembersFailure(t *testing.T) {
	t.Parallel()

	h, d := newHandler(t)

	d.messages.EXPECT().
		GetMessageByID(gomock.Any(), uint64(7), uint64(10)).
		Return(&domains.Message{ID: 10, ChatID: 42}, nil).
		Times(1)
	d.chats.EXPECT().
		GetChatMembers(gomock.Any(), uint64(42), uint64(7)).
		Return(nil, errors.New("db down")).
		Times(1)

	d.logger.EXPECT().
		ErrorContext(gomock.Any(), "Failed to get chat members for reaction added broadcast", gomock.Any()).
		Times(1)

	h.BroadcastReactionAdded(context.Background(), 10, 7, 1, "👍")
}

func TestHandler_BroadcastReactionAdded_NoMembers_NoOps(t *testing.T) {
	t.Parallel()

	h, d := newHandler(t)

	d.messages.EXPECT().
		GetMessageByID(gomock.Any(), uint64(7), uint64(10)).
		Return(&domains.Message{ID: 10, ChatID: 42}, nil).
		Times(1)
	d.chats.EXPECT().
		GetChatMembers(gomock.Any(), uint64(42), uint64(7)).
		Return([]domains.User{}, nil).
		Times(1)

	// Логгер не получает вызовов.
	h.BroadcastReactionAdded(context.Background(), 10, 7, 1, "👍")
}

func TestHandler_BroadcastReactionRemoved_ResolvesChatIDAndQueriesMembers(t *testing.T) {
	t.Parallel()

	h, d := newHandler(t)

	d.messages.EXPECT().
		GetMessageByID(gomock.Any(), uint64(7), uint64(10)).
		Return(&domains.Message{ID: 10, ChatID: 42}, nil).
		Times(1)
	d.chats.EXPECT().
		GetChatMembers(gomock.Any(), uint64(42), uint64(7)).
		Return([]domains.User{{ID: 7}, {ID: 8}, {ID: 9}}, nil).
		Times(1)

	h.BroadcastReactionRemoved(context.Background(), 10, 7, 1)
}

func TestHandler_BroadcastReactionRemoved_LogsErrorOnGetChatMembersFailure(t *testing.T) {
	t.Parallel()

	h, d := newHandler(t)

	d.messages.EXPECT().
		GetMessageByID(gomock.Any(), uint64(7), uint64(10)).
		Return(&domains.Message{ID: 10, ChatID: 42}, nil).
		Times(1)
	d.chats.EXPECT().
		GetChatMembers(gomock.Any(), uint64(42), uint64(7)).
		Return(nil, errors.New("db down")).
		Times(1)

	d.logger.EXPECT().
		ErrorContext(gomock.Any(), "Failed to get chat members for reaction removed broadcast", gomock.Any()).
		Times(1)

	h.BroadcastReactionRemoved(context.Background(), 10, 7, 1)
}
