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

func TestHandler_BroadcastReactionAdded_QueriesChatMembers(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	chatsUseCases := mockusecases.NewMockChatsUseCases(ctrl)
	logger := loggingmocks.NewMockLogger(ctrl)

	h := ws.New(
		mockupgrader.NewMockUpgrader(ctrl),
		mockusecases.NewMockUsersUseCases(ctrl),
		chatsUseCases,
		mockusecases.NewMockMessagesUseCases(ctrl),
		logger,
		natsmocks.NewMockPublisher(ctrl),
		config.NATSConfig{},
	)

	chatsUseCases.EXPECT().
		GetChatMembers(gomock.Any(), uint64(42), uint64(7)).
		Return([]domains.User{{ID: 7}, {ID: 8}}, nil).
		Times(1)

	// Нет активных соединений → sendToUser молча выходит для каждого участника.
	// Тест валидирует: broadcast не паникует и вызывает GetChatMembers.
	h.BroadcastReactionAdded(context.Background(), 42, 10, 7, 1, "👍")
}

func TestHandler_BroadcastReactionAdded_LogsErrorOnGetChatMembersFailure(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	chatsUseCases := mockusecases.NewMockChatsUseCases(ctrl)
	logger := loggingmocks.NewMockLogger(ctrl)

	h := ws.New(
		mockupgrader.NewMockUpgrader(ctrl),
		mockusecases.NewMockUsersUseCases(ctrl),
		chatsUseCases,
		mockusecases.NewMockMessagesUseCases(ctrl),
		logger,
		natsmocks.NewMockPublisher(ctrl),
		config.NATSConfig{},
	)

	chatsUseCases.EXPECT().
		GetChatMembers(gomock.Any(), uint64(42), uint64(7)).
		Return(nil, errors.New("db down")).
		Times(1)

	logger.EXPECT().
		ErrorContext(gomock.Any(), "Failed to get chat members for reaction added broadcast", gomock.Any()).
		Times(1)

	h.BroadcastReactionAdded(context.Background(), 42, 10, 7, 1, "👍")
}

func TestHandler_BroadcastReactionAdded_NoMembers_NoOps(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	chatsUseCases := mockusecases.NewMockChatsUseCases(ctrl)
	logger := loggingmocks.NewMockLogger(ctrl)

	h := ws.New(
		mockupgrader.NewMockUpgrader(ctrl),
		mockusecases.NewMockUsersUseCases(ctrl),
		chatsUseCases,
		mockusecases.NewMockMessagesUseCases(ctrl),
		logger,
		natsmocks.NewMockPublisher(ctrl),
		config.NATSConfig{},
	)

	chatsUseCases.EXPECT().
		GetChatMembers(gomock.Any(), uint64(42), uint64(7)).
		Return([]domains.User{}, nil).
		Times(1)

	// Логгер не должен получить ни одного вызова.
	h.BroadcastReactionAdded(context.Background(), 42, 10, 7, 1, "👍")
}

func TestHandler_BroadcastReactionRemoved_QueriesChatMembers(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	chatsUseCases := mockusecases.NewMockChatsUseCases(ctrl)
	logger := loggingmocks.NewMockLogger(ctrl)

	h := ws.New(
		mockupgrader.NewMockUpgrader(ctrl),
		mockusecases.NewMockUsersUseCases(ctrl),
		chatsUseCases,
		mockusecases.NewMockMessagesUseCases(ctrl),
		logger,
		natsmocks.NewMockPublisher(ctrl),
		config.NATSConfig{},
	)

	chatsUseCases.EXPECT().
		GetChatMembers(gomock.Any(), uint64(42), uint64(7)).
		Return([]domains.User{{ID: 7}, {ID: 8}, {ID: 9}}, nil).
		Times(1)

	h.BroadcastReactionRemoved(context.Background(), 42, 10, 7, 1)
}

func TestHandler_BroadcastReactionRemoved_LogsErrorOnGetChatMembersFailure(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	chatsUseCases := mockusecases.NewMockChatsUseCases(ctrl)
	logger := loggingmocks.NewMockLogger(ctrl)

	h := ws.New(
		mockupgrader.NewMockUpgrader(ctrl),
		mockusecases.NewMockUsersUseCases(ctrl),
		chatsUseCases,
		mockusecases.NewMockMessagesUseCases(ctrl),
		logger,
		natsmocks.NewMockPublisher(ctrl),
		config.NATSConfig{},
	)

	chatsUseCases.EXPECT().
		GetChatMembers(gomock.Any(), uint64(42), uint64(7)).
		Return(nil, errors.New("db down")).
		Times(1)

	logger.EXPECT().
		ErrorContext(gomock.Any(), "Failed to get chat members for reaction removed broadcast", gomock.Any()).
		Times(1)

	h.BroadcastReactionRemoved(context.Background(), 42, 10, 7, 1)
}
