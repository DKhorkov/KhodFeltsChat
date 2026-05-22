package notifications_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/services/notifications"
	mockrepositories "github.com/DKhorkov/kfc/mocks/repositories"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	service := notifications.New(
		mockrepositories.NewMockEmailsRepository(ctrl),
		mockrepositories.NewMockWebPushRepository(ctrl),
	)

	assert.NotNil(t, service)
}

func TestService_SendVerifyEmailMessage(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		user          domains.User
		setupMocks    func(*mockrepositories.MockEmailsRepository)
		expectedError error
	}{
		{
			name: "successful send verify email message",
			user: domains.User{
				ID:        1,
				Username:  "john_doe",
				Email:     "john@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(mockEmailsRepo *mockrepositories.MockEmailsRepository) {
				mockEmailsRepo.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), domains.User{
						ID:        1,
						Username:  "john_doe",
						Email:     "john@example.com",
						CreatedAt: now,
						UpdatedAt: now,
					}).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "repository returns error",
			user: domains.User{
				ID:        999,
				Username:  "nonexistent",
				Email:     "nonexistent@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(mockEmailsRepo *mockrepositories.MockEmailsRepository) {
				mockEmailsRepo.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), gomock.Any()).
					Return(errors.New("user not found"))
			},
			expectedError: errors.New("user not found"),
		},
		{
			name: "email service unavailable",
			user: domains.User{
				ID:        5,
				Username:  "user",
				Email:     "user@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(mockEmailsRepo *mockrepositories.MockEmailsRepository) {
				mockEmailsRepo.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), gomock.Any()).
					Return(errors.New("email service unavailable"))
			},
			expectedError: errors.New("email service unavailable"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockEmailsRepo := mockrepositories.NewMockEmailsRepository(ctrl)
			mockWebPushRepo := mockrepositories.NewMockWebPushRepository(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockEmailsRepo)
			}

			service := notifications.New(
				mockEmailsRepo,
				mockWebPushRepo,
			)

			ctx := context.Background()
			err := service.SendVerifyEmailMessage(ctx, tt.user)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_SendForgetPasswordMessage(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		user          domains.User
		setupMocks    func(*mockrepositories.MockEmailsRepository)
		expectedError error
	}{
		{
			name: "successful send forget password message",
			user: domains.User{
				ID:        1,
				Username:  "john_doe",
				Email:     "john@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(mockEmailsRepo *mockrepositories.MockEmailsRepository) {
				mockEmailsRepo.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), domains.User{
						ID:        1,
						Username:  "john_doe",
						Email:     "john@example.com",
						CreatedAt: now,
						UpdatedAt: now,
					}).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "repository returns error",
			user: domains.User{
				ID:        999,
				Username:  "nonexistent",
				Email:     "nonexistent@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(mockEmailsRepo *mockrepositories.MockEmailsRepository) {
				mockEmailsRepo.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), gomock.Any()).
					Return(errors.New("user not found"))
			},
			expectedError: errors.New("user not found"),
		},
		{
			name: "email service unavailable",
			user: domains.User{
				ID:        4,
				Username:  "user",
				Email:     "user@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(mockEmailsRepo *mockrepositories.MockEmailsRepository) {
				mockEmailsRepo.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), gomock.Any()).
					Return(errors.New("email service unavailable"))
			},
			expectedError: errors.New("email service unavailable"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockEmailsRepo := mockrepositories.NewMockEmailsRepository(ctrl)
			mockWebPushRepo := mockrepositories.NewMockWebPushRepository(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockEmailsRepo)
			}

			service := notifications.New(
				mockEmailsRepo,
				mockWebPushRepo,
			)

			ctx := context.Background()
			err := service.SendForgetPasswordMessage(ctx, tt.user)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_SendNewMessageByEmail(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		recipient     domains.User
		message       domains.Message
		chat          domains.Chat
		setupMocks    func(*mockrepositories.MockEmailsRepository)
		expectedError error
	}{
		{
			name: "successful send new message email",
			recipient: domains.User{
				ID:       1,
				Username: "john_doe",
				Email:    "john@example.com",
			},
			message: domains.Message{
				ID:     10,
				ChatID: 5,
				Text:   "Hello!",
				Sender: domains.User{
					ID:       2,
					Username: "alice",
				},
				CreatedAt: now,
			},
			chat: domains.Chat{
				ID: 5,
			},
			setupMocks: func(mockEmailsRepo *mockrepositories.MockEmailsRepository) {
				mockEmailsRepo.EXPECT().
					SendMessage(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "repository returns error",
			recipient: domains.User{
				ID:       1,
				Username: "john_doe",
				Email:    "john@example.com",
			},
			message: domains.Message{
				ID:     10,
				ChatID: 5,
				Text:   "Hello!",
			},
			chat: domains.Chat{
				ID: 5,
			},
			setupMocks: func(mockEmailsRepo *mockrepositories.MockEmailsRepository) {
				mockEmailsRepo.EXPECT().
					SendMessage(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("smtp error"))
			},
			expectedError: errors.New("smtp error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockEmailsRepo := mockrepositories.NewMockEmailsRepository(ctrl)
			mockWebPushRepo := mockrepositories.NewMockWebPushRepository(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockEmailsRepo)
			}

			service := notifications.New(
				mockEmailsRepo,
				mockWebPushRepo,
			)

			ctx := context.Background()
			err := service.SendNewMessageByEmail(ctx, tt.recipient, tt.message, tt.chat)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_SendNewMessageByWebPush(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		subscription  domains.WebPushSubscription
		message       domains.Message
		setupMocks    func(*mockrepositories.MockWebPushRepository)
		expectedError error
	}{
		{
			name: "successful send web push notification",
			subscription: domains.WebPushSubscription{
				ID:            1,
				UserID:        1,
				Endpoint:      "https://push.example.com/sub1",
				EncryptionKey: "p256dh_key",
				Auth:          "auth_key",
			},
			message: domains.Message{
				ID:     10,
				ChatID: 5,
				Text:   "Hello!",
				Sender: domains.User{
					ID:       2,
					Username: "alice",
				},
				CreatedAt: now,
			},
			setupMocks: func(mockWebPushRepo *mockrepositories.MockWebPushRepository) {
				mockWebPushRepo.EXPECT().
					SendNotification(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "repository returns error",
			subscription: domains.WebPushSubscription{
				ID:            1,
				UserID:        1,
				Endpoint:      "https://push.example.com/sub1",
				EncryptionKey: "p256dh_key",
				Auth:          "auth_key",
			},
			message: domains.Message{
				ID:     10,
				ChatID: 5,
				Text:   "Hello!",
			},
			setupMocks: func(mockWebPushRepo *mockrepositories.MockWebPushRepository) {
				mockWebPushRepo.EXPECT().
					SendNotification(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("web push error"))
			},
			expectedError: errors.New("web push error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockEmailsRepo := mockrepositories.NewMockEmailsRepository(ctrl)
			mockWebPushRepo := mockrepositories.NewMockWebPushRepository(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockWebPushRepo)
			}

			service := notifications.New(
				mockEmailsRepo,
				mockWebPushRepo,
			)

			ctx := context.Background()
			err := service.SendNewMessageByWebPush(ctx, tt.subscription, tt.message)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
