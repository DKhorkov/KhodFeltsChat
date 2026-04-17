package verify_email_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/workers/handlers/builders/verify_email"
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
		name     string
		useCases interfaces.NotificationsUseCases
		logger   logging.Logger
	}{
		{
			name:     "create builder with valid params",
			useCases: mockusecases.NewMockNotificationsUseCases(gomock.NewController(t)),
			logger:   mocks.NewMockLogger(gomock.NewController(t)),
		},
		{
			name:     "create builder with nil useCases",
			useCases: nil,
			logger:   mocks.NewMockLogger(gomock.NewController(t)),
		},
		{
			name:     "create builder with nil logger",
			useCases: mockusecases.NewMockNotificationsUseCases(gomock.NewController(t)),
			logger:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			builder := verify_email.New(tt.useCases, tt.logger)

			assert.NotNil(t, builder)
		})
	}
}

func TestBuilder_MessageHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupMocks  func(*mockusecases.MockNotificationsUseCases, *mocks.MockLogger)
		message     *nats.Msg
		expectError bool
	}{
		{
			name: "successful message handling",
			setupMocks: func(
				mockUseCases *mockusecases.MockNotificationsUseCases,
				_ *mocks.MockLogger,
			) {
				mockUseCases.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), uint64(1)).
					Return(nil)
			},
			message: func() *nats.Msg {
				dto := domains.VerifyEmailNotificationDTO{UserID: 1}
				data, _ := json.Marshal(dto)

				return &nats.Msg{Data: data}
			}(),
			expectError: false,
		},
		{
			name: "invalid JSON message",
			setupMocks: func(
				_ *mockusecases.MockNotificationsUseCases,
				mockLogger *mocks.MockLogger,
			) {
				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			message: &nats.Msg{
				Data: []byte("invalid json"),
			},
			expectError: false,
		},
		{
			name: "empty message data",
			setupMocks: func(
				_ *mockusecases.MockNotificationsUseCases,
				mockLogger *mocks.MockLogger,
			) {
				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			message: &nats.Msg{
				Data: []byte{},
			},
			expectError: false,
		},
		{
			name: "nil message data",
			setupMocks: func(
				_ *mockusecases.MockNotificationsUseCases,
				mockLogger *mocks.MockLogger,
			) {
				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			message: &nats.Msg{
				Data: nil,
			},
			expectError: false,
		},
		{
			name: "use case returns error",
			setupMocks: func(
				mockUseCases *mockusecases.MockNotificationsUseCases,
				mockLogger *mocks.MockLogger,
			) {
				mockUseCases.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), uint64(2)).
					Return(errors.New("user not found"))

				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			message: func() *nats.Msg {
				dto := domains.VerifyEmailNotificationDTO{UserID: 2}
				data, _ := json.Marshal(dto)

				return &nats.Msg{Data: data}
			}(),
			expectError: false,
		},
		{
			name: "message with large user id",
			setupMocks: func(
				mockUseCases *mockusecases.MockNotificationsUseCases,
				_ *mocks.MockLogger,
			) {
				mockUseCases.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), uint64(999999)).
					Return(nil)
			},
			message: func() *nats.Msg {
				dto := domains.VerifyEmailNotificationDTO{UserID: 999999}
				data, _ := json.Marshal(dto)

				return &nats.Msg{Data: data}
			}(),
			expectError: false,
		},
		{
			name: "message with zero user id",
			setupMocks: func(
				mockUseCases *mockusecases.MockNotificationsUseCases,
				mockLogger *mocks.MockLogger,
			) {
				mockUseCases.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), uint64(0)).
					Return(errors.New("invalid user id"))

				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			message: func() *nats.Msg {
				dto := domains.VerifyEmailNotificationDTO{UserID: 0}
				data, _ := json.Marshal(dto)

				return &nats.Msg{Data: data}
			}(),
			expectError: false,
		},
		{
			name: "email already confirmed",
			setupMocks: func(
				mockUseCases *mockusecases.MockNotificationsUseCases,
				mockLogger *mocks.MockLogger,
			) {
				mockUseCases.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), uint64(3)).
					Return(errors.New("email already confirmed"))

				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			message: func() *nats.Msg {
				dto := domains.VerifyEmailNotificationDTO{UserID: 3}
				data, _ := json.Marshal(dto)

				return &nats.Msg{Data: data}
			}(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockUseCases := mockusecases.NewMockNotificationsUseCases(ctrl)
			mockLogger := mocks.NewMockLogger(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockUseCases, mockLogger)
			}

			builder := verify_email.New(mockUseCases, mockLogger)

			ctx := context.Background()
			handler := builder.MessageHandler(ctx)

			assert.NotPanics(t, func() {
				handler(tt.message)
			})
		})
	}
}
