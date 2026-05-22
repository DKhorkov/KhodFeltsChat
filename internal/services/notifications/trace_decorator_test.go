package notifications_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/services/notifications"
	mockservices "github.com/DKhorkov/kfc/mocks/services"
	"github.com/DKhorkov/libs/tracing"
	mocktracing "github.com/DKhorkov/libs/tracing/mocks"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
)

func TestNewTraceDecorator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		traceProvider tracing.Provider
		spanConfig    tracing.SpanConfig
		base          interfaces.NotificationsService
	}{
		{
			name:          "create trace decorator with valid params",
			traceProvider: mocktracing.NewMockProvider(gomock.NewController(t)),
			spanConfig: tracing.SpanConfig{
				Name: "test-span",
				Opts: []trace.SpanStartOption{},
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{
						Name: "start",
						Opts: []trace.EventOption{},
					},
					End: tracing.SpanEventConfig{
						Name: "end",
						Opts: []trace.EventOption{},
					},
				},
			},
			base: mockservices.NewMockNotificationsService(gomock.NewController(t)),
		},
		{
			name:          "create trace decorator with nil base",
			traceProvider: mocktracing.NewMockProvider(gomock.NewController(t)),
			spanConfig:    tracing.SpanConfig{},
			base:          nil,
		},
		{
			name:          "create trace decorator with nil provider",
			traceProvider: nil,
			spanConfig:    tracing.SpanConfig{},
			base:          mockservices.NewMockNotificationsService(gomock.NewController(t)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decorator := notifications.NewTraceDecorator(tt.traceProvider, tt.spanConfig, tt.base)

			assert.NotNil(t, decorator)
		})
	}
}

func TestTraceDecorator_SendForgetPasswordMessage(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		user          domains.User
		setupMocks    func(*mocktracing.MockProvider, *mockservices.MockNotificationsService, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name: "successful send forget password message with tracing",
			user: domains.User{
				ID:        1,
				Username:  "john_doe",
				Email:     "john@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockNotificationsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
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
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockNotificationsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
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
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockNotificationsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
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
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockNotificationsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
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
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockNotificationsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
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

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockNotificationsService(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Opts: []trace.SpanStartOption{},
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := notifications.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			err := decorator.SendForgetPasswordMessage(ctx, tt.user)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTraceDecorator_SendNewMessageByEmail(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		recipient     domains.User
		message       domains.Message
		chat          domains.Chat
		setupMocks    func(*mocktracing.MockProvider, *mockservices.MockNotificationsService, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name: "successful send new message by email with tracing",
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
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockNotificationsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					SendNewMessageByEmail(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "error sending new message by email",
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
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockNotificationsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SendNewMessageByEmail(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("smtp error"))
			},
			expectedError: errors.New("smtp error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockNotificationsService(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Opts: []trace.SpanStartOption{},
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := notifications.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			err := decorator.SendNewMessageByEmail(ctx, tt.recipient, tt.message, tt.chat)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTraceDecorator_SendNewMessageByWebPush(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		subscription  domains.WebPushSubscription
		message       domains.Message
		setupMocks    func(*mocktracing.MockProvider, *mockservices.MockNotificationsService, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name: "successful send new message by web push with tracing",
			subscription: domains.WebPushSubscription{
				ID:            1,
				UserID:        1,
				Endpoint:      "https://push.example.com/sub1",
				EncryptionKey: "p256dh_key",
				Auth:          "auth_key",
				CreatedAt:     now,
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
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockNotificationsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					SendNewMessageByWebPush(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "error sending web push notification",
			subscription: domains.WebPushSubscription{
				ID:            1,
				UserID:        1,
				Endpoint:      "https://push.example.com/sub1",
				EncryptionKey: "p256dh_key",
				Auth:          "auth_key",
				CreatedAt:     now,
			},
			message: domains.Message{
				ID:     10,
				ChatID: 5,
				Text:   "Hello!",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockNotificationsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SendNewMessageByWebPush(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("web push error"))
			},
			expectedError: errors.New("web push error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockNotificationsService(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Opts: []trace.SpanStartOption{},
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := notifications.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			err := decorator.SendNewMessageByWebPush(ctx, tt.subscription, tt.message)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTraceDecorator_SendVerifyEmailMessage(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		user          domains.User
		setupMocks    func(*mocktracing.MockProvider, *mockservices.MockNotificationsService, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name: "successful send verify email message with tracing",
			user: domains.User{
				ID:        1,
				Username:  "john_doe",
				Email:     "john@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockNotificationsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
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
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockNotificationsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
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
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockNotificationsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
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
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockNotificationsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
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
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockNotificationsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
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
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockNotificationsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
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

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockNotificationsService(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Opts: []trace.SpanStartOption{},
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := notifications.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			err := decorator.SendVerifyEmailMessage(ctx, tt.user)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
