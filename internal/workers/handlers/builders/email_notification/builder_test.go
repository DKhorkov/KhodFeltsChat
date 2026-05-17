package email_notification_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/workers/handlers/builders/email_notification"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/logging"
	"github.com/DKhorkov/libs/logging/mocks"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var (
	testVerifyEmailMsg = func() *nats.Msg {
		dto := domains.EmailNotificationDTO{
			Type:    domains.EmailTypeVerifyEmail,
			UserID:  1,
			Payload: nil,
		}

		data, _ := json.Marshal(dto)

		return &nats.Msg{Data: data}
	}()

	testForgetPasswordMsg = func() *nats.Msg {
		dto := domains.EmailNotificationDTO{
			Type:    domains.EmailTypeForgetPassword,
			UserID:  2,
			Payload: nil,
		}

		data, _ := json.Marshal(dto)

		return &nats.Msg{Data: data}
	}()

	testNewMessageEmailMsg = func() *nats.Msg {
		payload, _ := json.Marshal(domains.NewMessagePayload{
			MessageID: 10,
		})

		dto := domains.EmailNotificationDTO{
			Type:    domains.EmailTypeNewMessage,
			UserID:  3,
			Payload: payload,
		}

		data, _ := json.Marshal(dto)

		return &nats.Msg{Data: data}
	}()
)

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

			builder := email_notification.New(
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
			name: "successful verify email handling",
			setupMocks: func(
				mockNotificationsUseCases *mockusecases.MockNotificationsUseCases,
				_ *mockusecases.MockSettingsUseCases,
				_ *mocks.MockLogger,
			) {
				mockNotificationsUseCases.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), uint64(1)).
					Return(nil)
			},
			message: testVerifyEmailMsg,
		},
		{
			name: "verify email returns error",
			setupMocks: func(
				mockNotificationsUseCases *mockusecases.MockNotificationsUseCases,
				_ *mockusecases.MockSettingsUseCases,
				mockLogger *mocks.MockLogger,
			) {
				mockNotificationsUseCases.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), uint64(1)).
					Return(errors.New("send failed"))

				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			message: testVerifyEmailMsg,
		},
		{
			name: "successful forget password handling",
			setupMocks: func(
				mockNotificationsUseCases *mockusecases.MockNotificationsUseCases,
				_ *mockusecases.MockSettingsUseCases,
				_ *mocks.MockLogger,
			) {
				mockNotificationsUseCases.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), uint64(2)).
					Return(nil)
			},
			message: testForgetPasswordMsg,
		},
		{
			name: "forget password returns error",
			setupMocks: func(
				mockNotificationsUseCases *mockusecases.MockNotificationsUseCases,
				_ *mockusecases.MockSettingsUseCases,
				mockLogger *mocks.MockLogger,
			) {
				mockNotificationsUseCases.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), uint64(2)).
					Return(errors.New("send failed"))

				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			message: testForgetPasswordMsg,
		},
		{
			name: "successful new message with consent",
			setupMocks: func(
				mockNotificationsUseCases *mockusecases.MockNotificationsUseCases,
				mockSettingsUseCases *mockusecases.MockSettingsUseCases,
				_ *mocks.MockLogger,
			) {
				mockSettingsUseCases.EXPECT().
					GetSettingsByUserID(gomock.Any(), uint64(3)).
					Return(&domains.Settings{
						EmailConsents: domains.ConsentNewMessage,
					}, nil)

				mockNotificationsUseCases.EXPECT().
					SendNewMessageByEmail(
						gomock.Any(),
						uint64(3),
						domains.NewMessagePayload{MessageID: 10},
					).
					Return(nil)
			},
			message: testNewMessageEmailMsg,
		},
		{
			name: "skip new message when no email consent",
			setupMocks: func(
				_ *mockusecases.MockNotificationsUseCases,
				mockSettingsUseCases *mockusecases.MockSettingsUseCases,
				_ *mocks.MockLogger,
			) {
				mockSettingsUseCases.EXPECT().
					GetSettingsByUserID(gomock.Any(), uint64(3)).
					Return(&domains.Settings{
						EmailConsents: 0,
					}, nil)
			},
			message: testNewMessageEmailMsg,
		},
		{
			name: "new message GetSettingsByUserID error",
			setupMocks: func(
				_ *mockusecases.MockNotificationsUseCases,
				mockSettingsUseCases *mockusecases.MockSettingsUseCases,
				mockLogger *mocks.MockLogger,
			) {
				mockSettingsUseCases.EXPECT().
					GetSettingsByUserID(gomock.Any(), uint64(3)).
					Return(nil, errors.New("settings error"))

				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			message: testNewMessageEmailMsg,
		},
		{
			name: "new message SendNewMessageByEmail error",
			setupMocks: func(
				mockNotificationsUseCases *mockusecases.MockNotificationsUseCases,
				mockSettingsUseCases *mockusecases.MockSettingsUseCases,
				mockLogger *mocks.MockLogger,
			) {
				mockSettingsUseCases.EXPECT().
					GetSettingsByUserID(gomock.Any(), uint64(3)).
					Return(&domains.Settings{
						EmailConsents: domains.ConsentNewMessage,
					}, nil)

				mockNotificationsUseCases.EXPECT().
					SendNewMessageByEmail(gomock.Any(), uint64(3), gomock.Any()).
					Return(errors.New("email send failed"))

				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			message: testNewMessageEmailMsg,
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
				dto := domains.EmailNotificationDTO{
					Type:   "unknown_type",
					UserID: 1,
				}
				data, _ := json.Marshal(dto)

				return &nats.Msg{Data: data}
			}(),
		},
		{
			name: "new message with invalid payload JSON",
			setupMocks: func(
				_ *mockusecases.MockNotificationsUseCases,
				mockSettingsUseCases *mockusecases.MockSettingsUseCases,
				mockLogger *mocks.MockLogger,
			) {
				mockSettingsUseCases.EXPECT().
					GetSettingsByUserID(gomock.Any(), uint64(3)).
					Return(&domains.Settings{
						EmailConsents: domains.ConsentNewMessage,
					}, nil)

				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			message: func() *nats.Msg {
				// Manually construct JSON with invalid payload field
				data := []byte(`{"type":"new_message","userId":3,"payload":"not an object"}`)

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

			builder := email_notification.New(
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
