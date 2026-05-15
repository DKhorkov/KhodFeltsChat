package web_push_notification_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/workers/handlers/builders/web_push_notification"
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
		name                         string
		webPushSubscriptionsUseCases interfaces.WebPushSubscriptionsUseCases
		messagesUseCases             interfaces.MessagesUseCases
		logger                       logging.Logger
	}{
		{
			name: "create builder with valid params",
			webPushSubscriptionsUseCases: mockusecases.NewMockWebPushSubscriptionsUseCases(
				gomock.NewController(t),
			),
			messagesUseCases: mockusecases.NewMockMessagesUseCases(
				gomock.NewController(t),
			),
			logger: mocks.NewMockLogger(gomock.NewController(t)),
		},
		{
			name:                         "create builder with nil webPushSubscriptionsUseCases",
			webPushSubscriptionsUseCases: nil,
			messagesUseCases: mockusecases.NewMockMessagesUseCases(
				gomock.NewController(t),
			),
			logger: mocks.NewMockLogger(gomock.NewController(t)),
		},
		{
			name: "create builder with nil messagesUseCases",
			webPushSubscriptionsUseCases: mockusecases.NewMockWebPushSubscriptionsUseCases(
				gomock.NewController(t),
			),
			messagesUseCases: nil,
			logger:           mocks.NewMockLogger(gomock.NewController(t)),
		},
		{
			name: "create builder with nil logger",
			webPushSubscriptionsUseCases: mockusecases.NewMockWebPushSubscriptionsUseCases(
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

			builder := web_push_notification.New(
				tt.webPushSubscriptionsUseCases,
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
			*mockusecases.MockWebPushSubscriptionsUseCases,
			*mockusecases.MockMessagesUseCases,
			*mocks.MockLogger,
		)
		message *nats.Msg
	}{
		{
			name: "successful message handling",
			setupMocks: func(
				mockPushUseCases *mockusecases.MockWebPushSubscriptionsUseCases,
				mockMsgUseCases *mockusecases.MockMessagesUseCases,
				_ *mocks.MockLogger,
			) {
				msg := &domains.Message{ID: 10, Text: "hello"}

				mockMsgUseCases.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
					Return(msg, nil)

				subs := []domains.WebPushSubscription{
					{ID: 1, UserID: 1, Endpoint: "https://push.example.com/sub1"},
					{ID: 2, UserID: 1, Endpoint: "https://push.example.com/sub2"},
				}

				mockPushUseCases.EXPECT().
					GetWebPushSubscriptionsByUserID(gomock.Any(), uint64(1)).
					Return(subs, nil)

				mockPushUseCases.EXPECT().
					SendWebPushNotification(gomock.Any(), subs[0], *msg).
					Return(nil)

				mockPushUseCases.EXPECT().
					SendWebPushNotification(gomock.Any(), subs[1], *msg).
					Return(nil)
			},
			message: func() *nats.Msg {
				dto := domains.WebPushNotificationDTO{UserID: 1, MessageID: 10}
				data, _ := json.Marshal(dto)

				return &nats.Msg{Data: data}
			}(),
		},
		{
			name: "invalid JSON message",
			setupMocks: func(
				_ *mockusecases.MockWebPushSubscriptionsUseCases,
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
				_ *mockusecases.MockWebPushSubscriptionsUseCases,
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
				dto := domains.WebPushNotificationDTO{UserID: 1, MessageID: 10}
				data, _ := json.Marshal(dto)

				return &nats.Msg{Data: data}
			}(),
		},
		{
			name: "GetWebPushSubscriptionsByUserID returns error",
			setupMocks: func(
				mockPushUseCases *mockusecases.MockWebPushSubscriptionsUseCases,
				mockMsgUseCases *mockusecases.MockMessagesUseCases,
				mockLogger *mocks.MockLogger,
			) {
				mockMsgUseCases.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
					Return(&domains.Message{ID: 10, Text: "hello"}, nil)

				mockPushUseCases.EXPECT().
					GetWebPushSubscriptionsByUserID(gomock.Any(), uint64(1)).
					Return(nil, errors.New("database error"))

				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			message: func() *nats.Msg {
				dto := domains.WebPushNotificationDTO{UserID: 1, MessageID: 10}
				data, _ := json.Marshal(dto)

				return &nats.Msg{Data: data}
			}(),
		},
		{
			name: "SendWebPushNotification returns error for one subscription",
			setupMocks: func(
				mockPushUseCases *mockusecases.MockWebPushSubscriptionsUseCases,
				mockMsgUseCases *mockusecases.MockMessagesUseCases,
				mockLogger *mocks.MockLogger,
			) {
				msg := &domains.Message{ID: 10, Text: "hello"}

				mockMsgUseCases.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
					Return(msg, nil)

				subs := []domains.WebPushSubscription{
					{ID: 1, UserID: 1, Endpoint: "https://push.example.com/sub1"},
					{ID: 2, UserID: 1, Endpoint: "https://push.example.com/sub2"},
				}

				mockPushUseCases.EXPECT().
					GetWebPushSubscriptionsByUserID(gomock.Any(), uint64(1)).
					Return(subs, nil)

				mockPushUseCases.EXPECT().
					SendWebPushNotification(gomock.Any(), subs[0], *msg).
					Return(errors.New("push service unavailable"))

				mockPushUseCases.EXPECT().
					SendWebPushNotification(gomock.Any(), subs[1], *msg).
					Return(nil)

				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			message: func() *nats.Msg {
				dto := domains.WebPushNotificationDTO{UserID: 1, MessageID: 10}
				data, _ := json.Marshal(dto)

				return &nats.Msg{Data: data}
			}(),
		},
		{
			name: "no subscriptions for user",
			setupMocks: func(
				mockPushUseCases *mockusecases.MockWebPushSubscriptionsUseCases,
				mockMsgUseCases *mockusecases.MockMessagesUseCases,
				_ *mocks.MockLogger,
			) {
				mockMsgUseCases.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
					Return(&domains.Message{ID: 10, Text: "hello"}, nil)

				mockPushUseCases.EXPECT().
					GetWebPushSubscriptionsByUserID(gomock.Any(), uint64(1)).
					Return([]domains.WebPushSubscription{}, nil)
			},
			message: func() *nats.Msg {
				dto := domains.WebPushNotificationDTO{UserID: 1, MessageID: 10}
				data, _ := json.Marshal(dto)

				return &nats.Msg{Data: data}
			}(),
		},
		{
			name: "empty message data",
			setupMocks: func(
				_ *mockusecases.MockWebPushSubscriptionsUseCases,
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
				_ *mockusecases.MockWebPushSubscriptionsUseCases,
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

			mockPushUseCases := mockusecases.NewMockWebPushSubscriptionsUseCases(ctrl)
			mockMsgUseCases := mockusecases.NewMockMessagesUseCases(ctrl)
			mockLogger := mocks.NewMockLogger(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockPushUseCases, mockMsgUseCases, mockLogger)
			}

			builder := web_push_notification.New(mockPushUseCases, mockMsgUseCases, mockLogger)

			ctx := context.Background()
			handler := builder.MessageHandler(ctx)

			assert.NotPanics(t, func() {
				handler(tt.message)
			})
		})
	}
}
