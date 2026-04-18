package notifications_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/services/notifications"
	mockrepositories "github.com/DKhorkov/kfc/mocks/repositories"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		emailsRepository interfaces.EmailsRepository
	}{
		{
			name:             "create notifications service with valid repository",
			emailsRepository: mockrepositories.NewMockEmailsRepository(gomock.NewController(t)),
		},
		{
			name:             "create notifications service with nil repository",
			emailsRepository: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := notifications.New(tt.emailsRepository)

			assert.NotNil(t, service)
		})
	}
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
			name: "user not found",
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
			name: "email already verified",
			user: domains.User{
				ID:        2,
				Username:  "verified_user",
				Email:     "verified@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(mockEmailsRepo *mockrepositories.MockEmailsRepository) {
				mockEmailsRepo.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), gomock.Any()).
					Return(errors.New("email already verified"))
			},
			expectedError: errors.New("email already verified"),
		},
		{
			name: "invalid email format",
			user: domains.User{
				ID:        3,
				Username:  "user",
				Email:     "invalid-email",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(mockEmailsRepo *mockrepositories.MockEmailsRepository) {
				mockEmailsRepo.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), gomock.Any()).
					Return(errors.New("invalid email format"))
			},
			expectedError: errors.New("invalid email format"),
		},
		{
			name: "rate limit exceeded",
			user: domains.User{
				ID:        4,
				Username:  "user",
				Email:     "user@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(mockEmailsRepo *mockrepositories.MockEmailsRepository) {
				mockEmailsRepo.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), gomock.Any()).
					Return(errors.New("rate limit exceeded, try again later"))
			},
			expectedError: errors.New("rate limit exceeded, try again later"),
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

			if tt.setupMocks != nil {
				tt.setupMocks(mockEmailsRepo)
			}

			service := notifications.New(mockEmailsRepo)

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
			name: "user not found",
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
			name: "invalid email format",
			user: domains.User{
				ID:        2,
				Username:  "user",
				Email:     "invalid-email",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(mockEmailsRepo *mockrepositories.MockEmailsRepository) {
				mockEmailsRepo.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), gomock.Any()).
					Return(errors.New("invalid email format"))
			},
			expectedError: errors.New("invalid email format"),
		},
		{
			name: "rate limit exceeded",
			user: domains.User{
				ID:        3,
				Username:  "user",
				Email:     "user@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(mockEmailsRepo *mockrepositories.MockEmailsRepository) {
				mockEmailsRepo.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), gomock.Any()).
					Return(errors.New("rate limit exceeded, try again later"))
			},
			expectedError: errors.New("rate limit exceeded, try again later"),
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
		{
			name: "empty email",
			user: domains.User{
				ID:        5,
				Username:  "user",
				Email:     "",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(mockEmailsRepo *mockrepositories.MockEmailsRepository) {
				mockEmailsRepo.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), gomock.Any()).
					Return(errors.New("email is required"))
			},
			expectedError: errors.New("email is required"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockEmailsRepo := mockrepositories.NewMockEmailsRepository(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockEmailsRepo)
			}

			service := notifications.New(mockEmailsRepo)

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
