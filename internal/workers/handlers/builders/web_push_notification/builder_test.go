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

var testNewMessageNatsMsg = func() *nats.Msg {
	payload, _ := json.Marshal(domains.NewMessagePayload{
		MessageID: 10,
	})

	dto := domains.WebPushNotificationDTO{
		Type:    domains.WebPushTypeNewMessage,
		UserID:  1,
		Payload: payload,
	}

	data, _ := json.Marshal(dto)

	return &nats.Msg{Data: data}
}()

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		notificationsUseCases interfaces.NotificationsUseCases
		settingsUseCases      interfaces.SettingsUseCases
		logger                logging.Logger
	}{
		{
			name: "create builder with valid params",
			notificationsUseCases: mockusecases.NewMockNotificationsUseCases(
				gomock.NewController(t),
			),
			settingsUseCases: mockusecases.NewMockSettingsUseCases(
				gomock.NewController(t),
			),
			logger: mocks.NewMockLogger(gomock.NewController(t)),
		},
		{
			name:                  "create builder with nil notificationsUseCases",
			notificationsUseCases: nil,
			settingsUseCases: mockusecases.NewMockSettingsUseCases(
				gomock.NewController(t),
			),
			logger: mocks.NewMockLogger(gomock.NewController(t)),
		},
		{
			name: "create builder with nil settingsUseCases",
			notificationsUseCases: mockusecases.NewMockNotificationsUseCases(
				gomock.NewController(t),
			),
			settingsUseCases: nil,
			logger:           mocks.NewMockLogger(gomock.NewController(t)),
		},
		{
			name: "create builder with nil logger",
			notificationsUseCases: mockusecases.NewMockNotificationsUseCases(
				gomock.NewController(t),
			),
			settingsUseCases: mockusecases.NewMockSettingsUseCases(
				gomock.NewController(t),
			),
			logger: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			builder := web_push_notification.New(
				tt.notificationsUseCases,
				tt.settingsUseCases,
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
			*mockusecases.MockNotificationsUseCases,
			*mockusecases.MockSettingsUseCases,
			*mocks.MockLogger,
		)
		message *nats.Msg
	}{
		{
			name: "successful message handling with consent",
			setupMocks: func(
				mockNotificationsUseCases *mockusecases.MockNotificationsUseCases,
				mockSettingsUseCases *mockusecases.MockSettingsUseCases,
				_ *mocks.MockLogger,
			) {
				mockSettingsUseCases.EXPECT().
					GetSettingsByUserID(gomock.Any(), uint64(1)).
					Return(&domains.Settings{
						WebPushConsents: domains.ConsentNewMessage,
					}, nil)

				mockNotificationsUseCases.EXPECT().
					SendNewMessageByWebPush(
						gomock.Any(),
						uint64(1),
						domains.NewMessagePayload{MessageID: 10},
					).
					Return(nil)
			},
			message: testNewMessageNatsMsg,
		},
		{
			name: "skip when no web push consent",
			setupMocks: func(
				_ *mockusecases.MockNotificationsUseCases,
				mockSettingsUseCases *mockusecases.MockSettingsUseCases,
				_ *mocks.MockLogger,
			) {
				mockSettingsUseCases.EXPECT().
					GetSettingsByUserID(gomock.Any(), uint64(1)).
					Return(&domains.Settings{
						WebPushConsents: 0,
					}, nil)
			},
			message: testNewMessageNatsMsg,
		},
		{
			name: "invalid JSON message",
			setupMocks: func(
				_ *mockusecases.MockNotificationsUseCases,
				_ *mockusecases.MockSettingsUseCases,
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
			name: "GetSettingsByUserID returns error",
			setupMocks: func(
				_ *mockusecases.MockNotificationsUseCases,
				mockSettingsUseCases *mockusecases.MockSettingsUseCases,
				mockLogger *mocks.MockLogger,
			) {
				mockSettingsUseCases.EXPECT().
					GetSettingsByUserID(gomock.Any(), uint64(1)).
					Return(nil, errors.New("settings error"))

				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			message: testNewMessageNatsMsg,
		},
		{
			name: "SendNewMessageByWebPush returns error",
			setupMocks: func(
				mockNotificationsUseCases *mockusecases.MockNotificationsUseCases,
				mockSettingsUseCases *mockusecases.MockSettingsUseCases,
				mockLogger *mocks.MockLogger,
			) {
				mockSettingsUseCases.EXPECT().
					GetSettingsByUserID(gomock.Any(), uint64(1)).
					Return(&domains.Settings{
						WebPushConsents: domains.ConsentNewMessage,
					}, nil)

				mockNotificationsUseCases.EXPECT().
					SendNewMessageByWebPush(gomock.Any(), uint64(1), gomock.Any()).
					Return(errors.New("push failed"))

				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			message: testNewMessageNatsMsg,
		},
		{
			name: "empty message data",
			setupMocks: func(
				_ *mockusecases.MockNotificationsUseCases,
				_ *mockusecases.MockSettingsUseCases,
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
				_ *mockusecases.MockNotificationsUseCases,
				_ *mockusecases.MockSettingsUseCases,
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
		{
			name: "unknown notification type",
			setupMocks: func(
				_ *mockusecases.MockNotificationsUseCases,
				_ *mockusecases.MockSettingsUseCases,
				mockLogger *mocks.MockLogger,
			) {
				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			message: func() *nats.Msg {
				dto := domains.WebPushNotificationDTO{
					Type:   "unknown_type",
					UserID: 1,
				}
				data, _ := json.Marshal(dto)

				return &nats.Msg{Data: data}
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockNotificationsUseCases := mockusecases.NewMockNotificationsUseCases(ctrl)
			mockSettingsUseCases := mockusecases.NewMockSettingsUseCases(ctrl)
			mockLogger := mocks.NewMockLogger(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockNotificationsUseCases, mockSettingsUseCases, mockLogger)
			}

			builder := web_push_notification.New(
				mockNotificationsUseCases,
				mockSettingsUseCases,
				mockLogger,
			)

			ctx := context.Background()
			handler := builder.MessageHandler(ctx)

			assert.NotPanics(t, func() {
				handler(tt.message)
			})
		})
	}
}
