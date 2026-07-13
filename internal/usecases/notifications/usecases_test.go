package notifications_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/usecases/notifications"
	mockservices "github.com/DKhorkov/kfc/mocks/services"
	"github.com/DKhorkov/libs/logging/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func newUseCases(
	ctrl *gomock.Controller,
	mockNotificationsService *mockservices.MockNotificationsService,
	mockUsersService *mockservices.MockUsersService,
	mockMessagesService *mockservices.MockMessagesService,
	mockChatsService *mockservices.MockChatsService,
	mockWebPushSubscriptionsService *mockservices.MockWebPushSubscriptionsService,
) *notifications.UseCases {
	return notifications.New(
		mockNotificationsService,
		mockUsersService,
		mockMessagesService,
		mockChatsService,
		mockWebPushSubscriptionsService,
		mocks.NewMockLogger(ctrl),
	)
}

func TestNew(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	uc := newUseCases(
		ctrl,
		mockservices.NewMockNotificationsService(ctrl),
		mockservices.NewMockUsersService(ctrl),
		mockservices.NewMockMessagesService(ctrl),
		mockservices.NewMockChatsService(ctrl),
		mockservices.NewMockWebPushSubscriptionsService(ctrl),
	)

	assert.NotNil(t, uc)
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

			uc := newUseCases(
				ctrl,
				mockNotificationsService,
				mockUsersService,
				mockservices.NewMockMessagesService(ctrl),
				mockservices.NewMockChatsService(ctrl),
				mockservices.NewMockWebPushSubscriptionsService(ctrl),
			)

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

			uc := newUseCases(
				ctrl,
				mockNotificationsService,
				mockUsersService,
				mockservices.NewMockMessagesService(ctrl),
				mockservices.NewMockChatsService(ctrl),
				mockservices.NewMockWebPushSubscriptionsService(ctrl),
			)

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

func TestUseCases_SendNewMessageByEmail(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name       string
		userID     uint64
		payload    domains.NewMessagePayload
		setupMocks func(
			*mockservices.MockUsersService,
			*mockservices.MockNotificationsService,
			*mockservices.MockMessagesService,
			*mockservices.MockChatsService,
		)
		expectedError error
	}{
		{
			name:   "successful send",
			userID: 1,
			payload: domains.NewMessagePayload{
				MessageID: 10,
			},
			setupMocks: func(
				mockUsersService *mockservices.MockUsersService,
				mockNotificationsService *mockservices.MockNotificationsService,
				mockMessagesService *mockservices.MockMessagesService,
				mockChatsService *mockservices.MockChatsService,
			) {
				user := &domains.User{
					ID:       1,
					Username: "john",
					Email:    "john@example.com",
				}
				mockUsersService.EXPECT().
					GetUserByID(gomock.Any(), uint64(1)).
					Return(user, nil)

				msg := &domains.Message{
					ID:     10,
					ChatID: 5,
					Text:   "Hello!",
					Sender: domains.User{ID: 2, Username: "alice"},
				}
				mockMessagesService.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
					Return(msg, nil)

				chat := &domains.Chat{
					ID:        5,
					CreatedAt: now,
				}
				mockChatsService.EXPECT().
					GetChatByID(gomock.Any(), uint64(5), uint64(1)).
					Return(chat, nil)

				mockNotificationsService.EXPECT().
					SendNewMessageByEmail(gomock.Any(), *user, *msg, *chat).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:   "user not found",
			userID: 999,
			payload: domains.NewMessagePayload{
				MessageID: 10,
			},
			setupMocks: func(
				mockUsersService *mockservices.MockUsersService,
				_ *mockservices.MockNotificationsService,
				_ *mockservices.MockMessagesService,
				_ *mockservices.MockChatsService,
			) {
				mockUsersService.EXPECT().
					GetUserByID(gomock.Any(), uint64(999)).
					Return(nil, errors.New("user not found"))
			},
			expectedError: errors.New("user not found"),
		},
		{
			name:   "message not found",
			userID: 1,
			payload: domains.NewMessagePayload{
				MessageID: 999,
			},
			setupMocks: func(
				mockUsersService *mockservices.MockUsersService,
				_ *mockservices.MockNotificationsService,
				mockMessagesService *mockservices.MockMessagesService,
				_ *mockservices.MockChatsService,
			) {
				mockUsersService.EXPECT().
					GetUserByID(gomock.Any(), uint64(1)).
					Return(&domains.User{ID: 1}, nil)

				mockMessagesService.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(999)).
					Return(nil, errors.New("message not found"))
			},
			expectedError: errors.New("message not found"),
		},
		{
			name:   "chat not found",
			userID: 1,
			payload: domains.NewMessagePayload{
				MessageID: 10,
			},
			setupMocks: func(
				mockUsersService *mockservices.MockUsersService,
				_ *mockservices.MockNotificationsService,
				mockMessagesService *mockservices.MockMessagesService,
				mockChatsService *mockservices.MockChatsService,
			) {
				mockUsersService.EXPECT().
					GetUserByID(gomock.Any(), uint64(1)).
					Return(&domains.User{ID: 1}, nil)

				mockMessagesService.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
					Return(&domains.Message{ID: 10, ChatID: 999}, nil)

				mockChatsService.EXPECT().
					GetChatByID(gomock.Any(), uint64(999), uint64(1)).
					Return(nil, errors.New("chat not found"))
			},
			expectedError: errors.New("chat not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockUsersService := mockservices.NewMockUsersService(ctrl)
			mockNotificationsService := mockservices.NewMockNotificationsService(ctrl)
			mockMessagesService := mockservices.NewMockMessagesService(ctrl)
			mockChatsService := mockservices.NewMockChatsService(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(
					mockUsersService,
					mockNotificationsService,
					mockMessagesService,
					mockChatsService,
				)
			}

			uc := newUseCases(
				ctrl,
				mockNotificationsService,
				mockUsersService,
				mockMessagesService,
				mockChatsService,
				mockservices.NewMockWebPushSubscriptionsService(ctrl),
			)

			ctx := context.Background()
			err := uc.SendNewMessageByEmail(ctx, tt.userID, tt.payload)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUseCases_SendNewMessageByWebPush(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     uint64
		payload    domains.NewMessagePayload
		setupMocks func(
			*mockservices.MockNotificationsService,
			*mockservices.MockMessagesService,
			*mockservices.MockWebPushSubscriptionsService,
			*mocks.MockLogger,
		)
		expectedError error
	}{
		{
			name:   "successful send to all subscriptions",
			userID: 1,
			payload: domains.NewMessagePayload{
				MessageID: 10,
			},
			setupMocks: func(
				mockNotificationsService *mockservices.MockNotificationsService,
				mockMessagesService *mockservices.MockMessagesService,
				mockWebPushService *mockservices.MockWebPushSubscriptionsService,
				_ *mocks.MockLogger,
			) {
				msg := &domains.Message{
					ID:     10,
					ChatID: 5,
					Text:   "Hello!",
					Sender: domains.User{ID: 2, Username: "alice"},
				}
				mockMessagesService.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
					Return(msg, nil)

				subs := []domains.WebPushSubscription{
					{ID: 1, UserID: 1, Endpoint: "https://push.example.com/sub1"},
					{ID: 2, UserID: 1, Endpoint: "https://push.example.com/sub2"},
				}
				mockWebPushService.EXPECT().
					GetWebPushSubscriptionsByUserID(gomock.Any(), uint64(1)).
					Return(subs, nil)

				mockMessagesService.EXPECT().
					GetUserUnreadCount(gomock.Any(), uint64(1)).
					Return(uint64(3), nil)

				mockNotificationsService.EXPECT().
					SendNewMessageByWebPush(gomock.Any(), subs[0], *msg, uint64(3)).
					Return(nil)

				mockNotificationsService.EXPECT().
					SendNewMessageByWebPush(gomock.Any(), subs[1], *msg, uint64(3)).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:   "message not found",
			userID: 1,
			payload: domains.NewMessagePayload{
				MessageID: 999,
			},
			setupMocks: func(
				_ *mockservices.MockNotificationsService,
				mockMessagesService *mockservices.MockMessagesService,
				_ *mockservices.MockWebPushSubscriptionsService,
				_ *mocks.MockLogger,
			) {
				mockMessagesService.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(999)).
					Return(nil, errors.New("message not found"))
			},
			expectedError: errors.New("message not found"),
		},
		{
			name:   "no subscriptions",
			userID: 1,
			payload: domains.NewMessagePayload{
				MessageID: 10,
			},
			setupMocks: func(
				_ *mockservices.MockNotificationsService,
				mockMessagesService *mockservices.MockMessagesService,
				mockWebPushService *mockservices.MockWebPushSubscriptionsService,
				_ *mocks.MockLogger,
			) {
				mockMessagesService.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
					Return(&domains.Message{ID: 10}, nil)

				mockWebPushService.EXPECT().
					GetWebPushSubscriptionsByUserID(gomock.Any(), uint64(1)).
					Return([]domains.WebPushSubscription{}, nil)

				mockMessagesService.EXPECT().
					GetUserUnreadCount(gomock.Any(), uint64(1)).
					Return(uint64(0), nil)
			},
			expectedError: nil,
		},
		{
			name:   "one subscription fails - continues to next",
			userID: 1,
			payload: domains.NewMessagePayload{
				MessageID: 10,
			},
			setupMocks: func(
				mockNotificationsService *mockservices.MockNotificationsService,
				mockMessagesService *mockservices.MockMessagesService,
				mockWebPushService *mockservices.MockWebPushSubscriptionsService,
				mockLogger *mocks.MockLogger,
			) {
				msg := &domains.Message{ID: 10, Text: "hello"}
				mockMessagesService.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
					Return(msg, nil)

				subs := []domains.WebPushSubscription{
					{ID: 1, UserID: 1, Endpoint: "https://push.example.com/sub1"},
					{ID: 2, UserID: 1, Endpoint: "https://push.example.com/sub2"},
				}
				mockWebPushService.EXPECT().
					GetWebPushSubscriptionsByUserID(gomock.Any(), uint64(1)).
					Return(subs, nil)

				mockMessagesService.EXPECT().
					GetUserUnreadCount(gomock.Any(), uint64(1)).
					Return(uint64(1), nil)

				mockNotificationsService.EXPECT().
					SendNewMessageByWebPush(gomock.Any(), subs[0], *msg, uint64(1)).
					Return(errors.New("push failed"))

				mockNotificationsService.EXPECT().
					SendNewMessageByWebPush(gomock.Any(), subs[1], *msg, uint64(1)).
					Return(nil)

				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any()).
					Times(1)
			},
			expectedError: nil,
		},
		{
			name:   "GetWebPushSubscriptionsByUserID error - propagated",
			userID: 1,
			payload: domains.NewMessagePayload{
				MessageID: 10,
			},
			setupMocks: func(
				_ *mockservices.MockNotificationsService,
				mockMessagesService *mockservices.MockMessagesService,
				mockWebPushService *mockservices.MockWebPushSubscriptionsService,
				_ *mocks.MockLogger,
			) {
				mockMessagesService.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
					Return(&domains.Message{ID: 10}, nil)

				mockWebPushService.EXPECT().
					GetWebPushSubscriptionsByUserID(gomock.Any(), uint64(1)).
					Return(nil, errors.New("db down"))
			},
			expectedError: errors.New("db down"),
		},
		{
			name:   "GetUserUnreadCount error - propagated",
			userID: 1,
			payload: domains.NewMessagePayload{
				MessageID: 10,
			},
			setupMocks: func(
				_ *mockservices.MockNotificationsService,
				mockMessagesService *mockservices.MockMessagesService,
				mockWebPushService *mockservices.MockWebPushSubscriptionsService,
				_ *mocks.MockLogger,
			) {
				mockMessagesService.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
					Return(&domains.Message{ID: 10}, nil)

				mockWebPushService.EXPECT().
					GetWebPushSubscriptionsByUserID(gomock.Any(), uint64(1)).
					Return([]domains.WebPushSubscription{{ID: 1}}, nil)

				mockMessagesService.EXPECT().
					GetUserUnreadCount(gomock.Any(), uint64(1)).
					Return(uint64(0), errors.New("counter unavailable"))
			},
			expectedError: errors.New("counter unavailable"),
		},
		{
			name:   "expired sub - DeleteWebPushSubscription error is logged, not propagated",
			userID: 1,
			payload: domains.NewMessagePayload{
				MessageID: 10,
			},
			setupMocks: func(
				mockNotificationsService *mockservices.MockNotificationsService,
				mockMessagesService *mockservices.MockMessagesService,
				mockWebPushService *mockservices.MockWebPushSubscriptionsService,
				mockLogger *mocks.MockLogger,
			) {
				msg := &domains.Message{ID: 10, Text: "hi"}
				mockMessagesService.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
					Return(msg, nil)

				subs := []domains.WebPushSubscription{{ID: 42, UserID: 1}}
				mockWebPushService.EXPECT().
					GetWebPushSubscriptionsByUserID(gomock.Any(), uint64(1)).
					Return(subs, nil)

				mockMessagesService.EXPECT().
					GetUserUnreadCount(gomock.Any(), uint64(1)).
					Return(uint64(5), nil)

				mockNotificationsService.EXPECT().
					SendNewMessageByWebPush(gomock.Any(), subs[0], *msg, uint64(5)).
					Return(customerrors.ErrWebPushSubscriptionExpired)

				mockWebPushService.EXPECT().
					DeleteWebPushSubscription(gomock.Any(), uint64(42)).
					Return(errors.New("delete failed"))

				// Info о том, что sub expired + Error о неудалённой подписке.
				mockLogger.EXPECT().Info(gomock.Any()).Times(1)
				mockLogger.EXPECT().Error(gomock.Any(), gomock.Any()).Times(1)
			},
			expectedError: nil, // ошибка удаления логируется, но не пробрасывается
		},
		{
			name:   "expired subscription is deleted",
			userID: 1,
			payload: domains.NewMessagePayload{
				MessageID: 10,
			},
			setupMocks: func(
				mockNotificationsService *mockservices.MockNotificationsService,
				mockMessagesService *mockservices.MockMessagesService,
				mockWebPushService *mockservices.MockWebPushSubscriptionsService,
				mockLogger *mocks.MockLogger,
			) {
				msg := &domains.Message{ID: 10, Text: "hello"}
				mockMessagesService.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
					Return(msg, nil)

				subs := []domains.WebPushSubscription{
					{ID: 1, UserID: 1, Endpoint: "https://push.example.com/sub1"},
				}
				mockWebPushService.EXPECT().
					GetWebPushSubscriptionsByUserID(gomock.Any(), uint64(1)).
					Return(subs, nil)

				mockMessagesService.EXPECT().
					GetUserUnreadCount(gomock.Any(), uint64(1)).
					Return(uint64(5), nil)

				mockNotificationsService.EXPECT().
					SendNewMessageByWebPush(gomock.Any(), subs[0], *msg, uint64(5)).
					Return(customerrors.ErrWebPushSubscriptionExpired)

				mockWebPushService.EXPECT().
					DeleteWebPushSubscription(gomock.Any(), uint64(1)).
					Return(nil)

				mockLogger.EXPECT().
					Info(gomock.Any()).
					Times(1)
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockNotificationsService := mockservices.NewMockNotificationsService(ctrl)
			mockMessagesService := mockservices.NewMockMessagesService(ctrl)
			mockWebPushService := mockservices.NewMockWebPushSubscriptionsService(ctrl)
			mockLogger := mocks.NewMockLogger(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(
					mockNotificationsService,
					mockMessagesService,
					mockWebPushService,
					mockLogger,
				)
			}

			uc := notifications.New(
				mockNotificationsService,
				mockservices.NewMockUsersService(ctrl),
				mockMessagesService,
				mockservices.NewMockChatsService(ctrl),
				mockWebPushService,
				mockLogger,
			)

			ctx := context.Background()
			err := uc.SendNewMessageByWebPush(ctx, tt.userID, tt.payload)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
