package emails_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/repositories/emails"
	mockrepositories "github.com/DKhorkov/kfc/mocks/repositories"
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
		base          interfaces.EmailsRepository
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
			base: mockrepositories.NewMockEmailsRepository(gomock.NewController(t)),
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
			base:          mockrepositories.NewMockEmailsRepository(gomock.NewController(t)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decorator := emails.NewTraceDecorator(tt.traceProvider, tt.spanConfig, tt.base)

			assert.NotNil(t, decorator)
		})
	}
}

func TestTraceDecorator_SendVerifyEmailMessage(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		user          domains.User
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockEmailsRepository, *mocktracing.MockSpan)
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
				mockBase *mockrepositories.MockEmailsRepository,
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
			name: "send verify email to user with unicode in username",
			user: domains.User{
				ID:        2,
				Username:  "пользователь",
				Email:     "user@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockEmailsRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), domains.User{
						ID:        2,
						Username:  "пользователь",
						Email:     "user@example.com",
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
				mockBase *mockrepositories.MockEmailsRepository,
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
				ID:        1,
				Username:  "verified_user",
				Email:     "verified@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockEmailsRepository,
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
				ID:        1,
				Username:  "user",
				Email:     "invalid-email",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockEmailsRepository,
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
				ID:        1,
				Username:  "user",
				Email:     "user@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockEmailsRepository,
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
				ID:        1,
				Username:  "user",
				Email:     "user@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockEmailsRepository,
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
			mockBase := mockrepositories.NewMockEmailsRepository(ctrl)
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

			decorator := emails.NewTraceDecorator(mockProvider, spanConfig, mockBase)

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

func TestTraceDecorator_SendForgetPasswordMessage(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		user          domains.User
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockEmailsRepository, *mocktracing.MockSpan)
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
				mockBase *mockrepositories.MockEmailsRepository,
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
			name: "send forget password message to user with unicode email",
			user: domains.User{
				ID:        2,
				Username:  "user",
				Email:     "user@почта.рф",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockEmailsRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), domains.User{
						ID:        2,
						Username:  "user",
						Email:     "user@почта.рф",
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
				mockBase *mockrepositories.MockEmailsRepository,
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
				ID:        1,
				Username:  "user",
				Email:     "invalid-email",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockEmailsRepository,
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
				ID:        1,
				Username:  "user",
				Email:     "user@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockEmailsRepository,
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
				ID:        1,
				Username:  "user",
				Email:     "user@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockEmailsRepository,
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
		{
			name: "user with empty email",
			user: domains.User{
				ID:        1,
				Username:  "user",
				Email:     "",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockEmailsRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
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

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockEmailsRepository(ctrl)
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

			decorator := emails.NewTraceDecorator(mockProvider, spanConfig, mockBase)

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
