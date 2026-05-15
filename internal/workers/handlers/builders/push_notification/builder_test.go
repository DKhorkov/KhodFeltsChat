package push_notification_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/workers/handlers/builders/push_notification"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/logging"
	"github.com/DKhorkov/libs/logging/mocks"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                      string
		pushSubscriptionsUseCases interfaces.PushSubscriptionsUseCases
		messagesUseCases          interfaces.MessagesUseCases
		logger                    logging.Logger
	}{
		{
			name: "create builder with valid params",
			pushSubscriptionsUseCases: mockusecases.NewMockPushSubscriptionsUseCases(
				gomock.NewController(t),
			),
			messagesUseCases: mockusecases.NewMockMessagesUseCases(
				gomock.NewController(t),
			),
			logger: mocks.NewMockLogger(gomock.NewController(t)),
		},
		{
			name:                      "create builder with nil pushSubscriptionsUseCases",
			pushSubscriptionsUseCases: nil,
			messagesUseCases: mockusecases.NewMockMessagesUseCases(
				gomock.NewController(t),
			),
			logger: mocks.NewMockLogger(gomock.NewController(t)),
		},
		{
			name: "create builder with nil messagesUseCases",
			pushSubscriptionsUseCases: mockusecases.NewMockPushSubscriptionsUseCases(
				gomock.NewController(t),
			),
			messagesUseCases: nil,
			logger:           mocks.NewMockLogger(gomock.NewController(t)),
		},
		{
			name: "create builder with nil logger",
			pushSubscriptionsUseCases: mockusecases.NewMockPushSubscriptionsUseCases(
				gomock.NewController(t),
			),
			messagesUseCases: mockusecases.NewMockMessagesUseCases(
				gomock.NewController(t),
			),
			logger: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			builder := push_notification.New(
				tt.pushSubscriptionsUseCases,
				tt.messagesUseCases,
				tt.logger,
			)

			assert.NotNil(t, builder)
		})
	}
}

func TestBuilder_MessageHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMocks func(
			*mockusecases.MockPushSubscriptionsUseCases,
			*mockusecases.MockMessagesUseCases,
			*mocks.MockLogger,
		)
		message *nats.Msg
	}{
		{
			name: "successful message handling",
			setupMocks: func(
				mockPushUseCases *mockusecases.MockPushSubscriptionsUseCases,
				mockMsgUseCases *mockusecases.MockMessagesUseCases,
				_ *mocks.MockLogger,
			) {
				msg := &domains.Message{ID: 10, Text: "hello"}

				mockMsgUseCases.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
					Return(msg, nil)

				subs := []domains.PushSubscription{
					{ID: 1, UserID: 1, Endpoint: "https://push.example.com/sub1"},
					{ID: 2, UserID: 1, Endpoint: "https://push.example.com/sub2"},
				}

				mockPushUseCases.EXPECT().
					GetPushSubscriptionsByUserID(gomock.Any(), uint64(1)).
					Return(subs, nil)

				mockPushUseCases.EXPECT().
					SendPushNotification(gomock.Any(), subs[0], *msg).
					Return(nil)

				mockPushUseCases.EXPECT().
					SendPushNotification(gomock.Any(), subs[1], *msg).
					Return(nil)
			},
			message: func() *nats.Msg {
				dto := domains.PushNotificationDTO{UserID: 1, MessageID: 10}
				data, _ := json.Marshal(dto)

				return &nats.Msg{Data: data}
			}(),
		},
		{
			name: "invalid JSON message",
			setupMocks: func(
				_ *mockusecases.MockPushSubscriptionsUseCases,
				_ *mockusecases.MockMessagesUseCases,
				mockLogger *mocks.MockLogger,
			) {
				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			message: &nats.Msg{
				Data: []byte("invalid json"),
			},
		},
		{
			name: "GetMessageByID returns error",
			setupMocks: func(
				_ *mockusecases.MockPushSubscriptionsUseCases,
				mockMsgUseCases *mockusecases.MockMessagesUseCases,
				mockLogger *mocks.MockLogger,
			) {
				mockMsgUseCases.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
					Return(nil, errors.New("message not found"))

				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			message: func() *nats.Msg {
				dto := domains.PushNotificationDTO{UserID: 1, MessageID: 10}
				data, _ := json.Marshal(dto)

				return &nats.Msg{Data: data}
			}(),
		},
		{
			name: "GetPushSubscriptionsByUserID returns error",
			setupMocks: func(
				mockPushUseCases *mockusecases.MockPushSubscriptionsUseCases,
				mockMsgUseCases *mockusecases.MockMessagesUseCases,
				mockLogger *mocks.MockLogger,
			) {
				mockMsgUseCases.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
					Return(&domains.Message{ID: 10, Text: "hello"}, nil)

				mockPushUseCases.EXPECT().
					GetPushSubscriptionsByUserID(gomock.Any(), uint64(1)).
					Return(nil, errors.New("database error"))

				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			message: func() *nats.Msg {
				dto := domains.PushNotificationDTO{UserID: 1, MessageID: 10}
				data, _ := json.Marshal(dto)

				return &nats.Msg{Data: data}
			}(),
		},
		{
			name: "SendPushNotification returns error for one subscription",
			setupMocks: func(
				mockPushUseCases *mockusecases.MockPushSubscriptionsUseCases,
				mockMsgUseCases *mockusecases.MockMessagesUseCases,
				mockLogger *mocks.MockLogger,
			) {
				msg := &domains.Message{ID: 10, Text: "hello"}

				mockMsgUseCases.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
					Return(msg, nil)

				subs := []domains.PushSubscription{
					{ID: 1, UserID: 1, Endpoint: "https://push.example.com/sub1"},
					{ID: 2, UserID: 1, Endpoint: "https://push.example.com/sub2"},
				}

				mockPushUseCases.EXPECT().
					GetPushSubscriptionsByUserID(gomock.Any(), uint64(1)).
					Return(subs, nil)

				mockPushUseCases.EXPECT().
					SendPushNotification(gomock.Any(), subs[0], *msg).
					Return(errors.New("push service unavailable"))

				mockPushUseCases.EXPECT().
					SendPushNotification(gomock.Any(), subs[1], *msg).
					Return(nil)

				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			message: func() *nats.Msg {
				dto := domains.PushNotificationDTO{UserID: 1, MessageID: 10}
				data, _ := json.Marshal(dto)

				return &nats.Msg{Data: data}
			}(),
		},
		{
			name: "no subscriptions for user",
			setupMocks: func(
				mockPushUseCases *mockusecases.MockPushSubscriptionsUseCases,
				mockMsgUseCases *mockusecases.MockMessagesUseCases,
				_ *mocks.MockLogger,
			) {
				mockMsgUseCases.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
					Return(&domains.Message{ID: 10, Text: "hello"}, nil)

				mockPushUseCases.EXPECT().
					GetPushSubscriptionsByUserID(gomock.Any(), uint64(1)).
					Return([]domains.PushSubscription{}, nil)
			},
			message: func() *nats.Msg {
				dto := domains.PushNotificationDTO{UserID: 1, MessageID: 10}
				data, _ := json.Marshal(dto)

				return &nats.Msg{Data: data}
			}(),
		},
		{
			name: "empty message data",
			setupMocks: func(
				_ *mockusecases.MockPushSubscriptionsUseCases,
				_ *mockusecases.MockMessagesUseCases,
				mockLogger *mocks.MockLogger,
			) {
				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			message: &nats.Msg{
				Data: []byte{},
			},
		},
		{
			name: "nil message data",
			setupMocks: func(
				_ *mockusecases.MockPushSubscriptionsUseCases,
				_ *mockusecases.MockMessagesUseCases,
				mockLogger *mocks.MockLogger,
			) {
				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			message: &nats.Msg{
				Data: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockPushUseCases := mockusecases.NewMockPushSubscriptionsUseCases(ctrl)
			mockMsgUseCases := mockusecases.NewMockMessagesUseCases(ctrl)
			mockLogger := mocks.NewMockLogger(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockPushUseCases, mockMsgUseCases, mockLogger)
			}

			builder := push_notification.New(mockPushUseCases, mockMsgUseCases, mockLogger)

			ctx := context.Background()
			handler := builder.MessageHandler(ctx)

			assert.NotPanics(t, func() {
				handler(tt.message)
			})
		})
	}
}
