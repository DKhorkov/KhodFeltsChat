package notifications_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/usecases/notifications"
	mockservices "github.com/DKhorkov/kfc/mocks/services"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		notificationsService interfaces.NotificationsService
		usersService         interfaces.UsersService
	}{
		{
			name:                 "create notifications usecases with valid services",
			notificationsService: mockservices.NewMockNotificationsService(gomock.NewController(t)),
			usersService:         mockservices.NewMockUsersService(gomock.NewController(t)),
		},
		{
			name:                 "create notifications usecases with nil services",
			notificationsService: nil,
			usersService:         nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := notifications.New(tt.notificationsService, tt.usersService)

			assert.NotNil(t, uc)
		})
	}
}

func TestUseCases_SendVerifyEmailMessage(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		userID        uint64
		setupMocks    func(*mockservices.MockUsersService, *mockservices.MockNotificationsService)
		expectedError error
	}{
		{
			name:   "successful send verify email message",
			userID: 1,
			setupMocks: func(
				mockUsersService *mockservices.MockUsersService,
				mockNotificationsService *mockservices.MockNotificationsService,
			) {
				mockUsersService.EXPECT().
					GetUserByID(gomock.Any(), uint64(1)).
					Return(&domains.User{
						ID:             1,
						Username:       "john_doe",
						Email:          "john@example.com",
						EmailConfirmed: false,
						CreatedAt:      now,
						UpdatedAt:      now,
					}, nil)

				mockNotificationsService.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), domains.User{
						ID:             1,
						Username:       "john_doe",
						Email:          "john@example.com",
						EmailConfirmed: false,
						CreatedAt:      now,
						UpdatedAt:      now,
					}).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:   "user not found",
			userID: 999,
			setupMocks: func(
				mockUsersService *mockservices.MockUsersService,
				_ *mockservices.MockNotificationsService,
			) {
				mockUsersService.EXPECT().
					GetUserByID(gomock.Any(), uint64(999)).
					Return(nil, errors.New("user not found"))
			},
			expectedError: errors.New("user not found"),
		},
		{
			name:   "email already confirmed",
			userID: 2,
			setupMocks: func(
				mockUsersService *mockservices.MockUsersService,
				_ *mockservices.MockNotificationsService,
			) {
				mockUsersService.EXPECT().
					GetUserByID(gomock.Any(), uint64(2)).
					Return(&domains.User{
						ID:             2,
						Username:       "verified_user",
						Email:          "verified@example.com",
						EmailConfirmed: true,
						CreatedAt:      now,
						UpdatedAt:      now,
					}, nil)
			},
			expectedError: customerrors.ErrEmailAlreadyConfirmed,
		},
		{
			name:   "notification service returns error",
			userID: 3,
			setupMocks: func(
				mockUsersService *mockservices.MockUsersService,
				mockNotificationsService *mockservices.MockNotificationsService,
			) {
				mockUsersService.EXPECT().
					GetUserByID(gomock.Any(), uint64(3)).
					Return(&domains.User{
						ID:             3,
						Username:       "user",
						Email:          "user@example.com",
						EmailConfirmed: false,
						CreatedAt:      now,
						UpdatedAt:      now,
					}, nil)

				mockNotificationsService.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), gomock.Any()).
					Return(errors.New("email service unavailable"))
			},
			expectedError: errors.New("email service unavailable"),
		},
		{
			name:   "user has empty email",
			userID: 4,
			setupMocks: func(
				mockUsersService *mockservices.MockUsersService,
				mockNotificationsService *mockservices.MockNotificationsService,
			) {
				mockUsersService.EXPECT().
					GetUserByID(gomock.Any(), uint64(4)).
					Return(&domains.User{
						ID:             4,
						Username:       "user",
						Email:          "",
						EmailConfirmed: false,
						CreatedAt:      now,
						UpdatedAt:      now,
					}, nil)

				mockNotificationsService.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), gomock.Any()).
					Return(errors.New("invalid email format"))
			},
			expectedError: errors.New("invalid email format"),
		},
		{
			name:   "rate limit exceeded",
			userID: 5,
			setupMocks: func(
				mockUsersService *mockservices.MockUsersService,
				mockNotificationsService *mockservices.MockNotificationsService,
			) {
				mockUsersService.EXPECT().
					GetUserByID(gomock.Any(), uint64(5)).
					Return(&domains.User{
						ID:             5,
						Username:       "user",
						Email:          "user@example.com",
						EmailConfirmed: false,
						CreatedAt:      now,
						UpdatedAt:      now,
					}, nil)

				mockNotificationsService.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), gomock.Any()).
					Return(errors.New("rate limit exceeded, try again later"))
			},
			expectedError: errors.New("rate limit exceeded, try again later"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockUsersService := mockservices.NewMockUsersService(ctrl)
			mockNotificationsService := mockservices.NewMockNotificationsService(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockUsersService, mockNotificationsService)
			}

			uc := notifications.New(mockNotificationsService, mockUsersService)

			ctx := context.Background()
			err := uc.SendVerifyEmailMessage(ctx, tt.userID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUseCases_SendForgetPasswordMessage(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		userID        uint64
		setupMocks    func(*mockservices.MockUsersService, *mockservices.MockNotificationsService)
		expectedError error
	}{
		{
			name:   "successful send forget password message",
			userID: 1,
			setupMocks: func(
				mockUsersService *mockservices.MockUsersService,
				mockNotificationsService *mockservices.MockNotificationsService,
			) {
				mockUsersService.EXPECT().
					GetUserByID(gomock.Any(), uint64(1)).
					Return(&domains.User{
						ID:             1,
						Username:       "john_doe",
						Email:          "john@example.com",
						EmailConfirmed: true,
						CreatedAt:      now,
						UpdatedAt:      now,
					}, nil)

				mockNotificationsService.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), domains.User{
						ID:             1,
						Username:       "john_doe",
						Email:          "john@example.com",
						EmailConfirmed: true,
						CreatedAt:      now,
						UpdatedAt:      now,
					}).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:   "user not found",
			userID: 999,
			setupMocks: func(
				mockUsersService *mockservices.MockUsersService,
				_ *mockservices.MockNotificationsService,
			) {
				mockUsersService.EXPECT().
					GetUserByID(gomock.Any(), uint64(999)).
					Return(nil, errors.New("user not found"))
			},
			expectedError: errors.New("user not found"),
		},
		{
			name:   "email not confirmed",
			userID: 2,
			setupMocks: func(
				mockUsersService *mockservices.MockUsersService,
				_ *mockservices.MockNotificationsService,
			) {
				mockUsersService.EXPECT().
					GetUserByID(gomock.Any(), uint64(2)).
					Return(&domains.User{
						ID:             2,
						Username:       "unverified_user",
						Email:          "unverified@example.com",
						EmailConfirmed: false,
						CreatedAt:      now,
						UpdatedAt:      now,
					}, nil)
			},
			expectedError: customerrors.ErrEmailNotConfirmed,
		},
		{
			name:   "notification service returns error",
			userID: 3,
			setupMocks: func(
				mockUsersService *mockservices.MockUsersService,
				mockNotificationsService *mockservices.MockNotificationsService,
			) {
				mockUsersService.EXPECT().
					GetUserByID(gomock.Any(), uint64(3)).
					Return(&domains.User{
						ID:             3,
						Username:       "user",
						Email:          "user@example.com",
						EmailConfirmed: true,
						CreatedAt:      now,
						UpdatedAt:      now,
					}, nil)

				mockNotificationsService.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), gomock.Any()).
					Return(errors.New("email service unavailable"))
			},
			expectedError: errors.New("email service unavailable"),
		},
		{
			name:   "user has empty email",
			userID: 4,
			setupMocks: func(
				mockUsersService *mockservices.MockUsersService,
				mockNotificationsService *mockservices.MockNotificationsService,
			) {
				mockUsersService.EXPECT().
					GetUserByID(gomock.Any(), uint64(4)).
					Return(&domains.User{
						ID:             4,
						Username:       "user",
						Email:          "",
						EmailConfirmed: true,
						CreatedAt:      now,
						UpdatedAt:      now,
					}, nil)

				mockNotificationsService.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), gomock.Any()).
					Return(errors.New("invalid email format"))
			},
			expectedError: errors.New("invalid email format"),
		},
		{
			name:   "rate limit exceeded",
			userID: 5,
			setupMocks: func(
				mockUsersService *mockservices.MockUsersService,
				mockNotificationsService *mockservices.MockNotificationsService,
			) {
				mockUsersService.EXPECT().
					GetUserByID(gomock.Any(), uint64(5)).
					Return(&domains.User{
						ID:             5,
						Username:       "user",
						Email:          "user@example.com",
						EmailConfirmed: true,
						CreatedAt:      now,
						UpdatedAt:      now,
					}, nil)

				mockNotificationsService.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), gomock.Any()).
					Return(errors.New("rate limit exceeded, try again later"))
			},
			expectedError: errors.New("rate limit exceeded, try again later"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockUsersService := mockservices.NewMockUsersService(ctrl)
			mockNotificationsService := mockservices.NewMockNotificationsService(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockUsersService, mockNotificationsService)
			}

			uc := notifications.New(mockNotificationsService, mockUsersService)

			ctx := context.Background()
			err := uc.SendForgetPasswordMessage(ctx, tt.userID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
