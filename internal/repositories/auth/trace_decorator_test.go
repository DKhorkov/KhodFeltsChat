package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/repositories/auth"
	mockrepositories "github.com/DKhorkov/kfc/mocks/repositories"
	"github.com/DKhorkov/libs/tracing"
	mocktracing "github.com/DKhorkov/libs/tracing/mocks"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
)

func TestNewTraceDecorator(t *testing.T) {
	tests := []struct {
		name          string
		traceProvider tracing.Provider
		spanConfig    tracing.SpanConfig
		base          interfaces.AuthRepository
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
			base: mockrepositories.NewMockAuthRepository(gomock.NewController(t)),
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
			base:          mockrepositories.NewMockAuthRepository(gomock.NewController(t)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decorator := auth.NewTraceDecorator(tt.traceProvider, tt.spanConfig, tt.base)

			assert.NotNil(t, decorator)
		})
	}
}

func TestTraceDecorator_RegisterUser(t *testing.T) {
	tests := []struct {
		name          string
		userData      domains.RegisterDTO
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockAuthRepository, *mocktracing.MockSpan)
		expectedID    uint64
		expectedError error
	}{
		{
			name: "successful registration with tracing",
			userData: domains.RegisterDTO{
				Username: "newuser",
				Email:    "newuser@example.com",
				Password: "Password123!",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					RegisterUser(gomock.Any(), domains.RegisterDTO{
						Username: "newuser",
						Email:    "newuser@example.com",
						Password: "Password123!",
					}).
					Return(uint64(1), nil)
			},
			expectedID:    1,
			expectedError: nil,
		},
		{
			name: "registration with existing email",
			userData: domains.RegisterDTO{
				Username: "existing",
				Email:    "existing@example.com",
				Password: "Password123!",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					RegisterUser(gomock.Any(), gomock.Any()).
					Return(uint64(0), errors.New("email already exists"))
			},
			expectedID:    0,
			expectedError: errors.New("email already exists"),
		},
		{
			name: "registration with weak password",
			userData: domains.RegisterDTO{
				Username: "weakuser",
				Email:    "weak@example.com",
				Password: "weak",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					RegisterUser(gomock.Any(), gomock.Any()).
					Return(uint64(0), errors.New("password does not meet security requirements"))
			},
			expectedID:    0,
			expectedError: errors.New("password does not meet security requirements"),
		},
		{
			name: "database error",
			userData: domains.RegisterDTO{
				Username: "user",
				Email:    "user@example.com",
				Password: "Password123!",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					RegisterUser(gomock.Any(), gomock.Any()).
					Return(uint64(0), errors.New("database connection failed"))
			},
			expectedID:    0,
			expectedError: errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockAuthRepository(ctrl)
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

			decorator := auth.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			id, err := decorator.RegisterUser(ctx, tt.userData)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedID, id)
		})
	}
}

func TestTraceDecorator_CreateRefreshToken(t *testing.T) {
	ttl := 24 * time.Hour

	tests := []struct {
		name          string
		userID        uint64
		refreshToken  string
		ttl           time.Duration
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockAuthRepository, *mocktracing.MockSpan)
		expectedID    uint64
		expectedError error
	}{
		{
			name:         "successful create refresh token with tracing",
			userID:       1,
			refreshToken: "refresh_token_123",
			ttl:          ttl,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					CreateRefreshToken(gomock.Any(), uint64(1), "refresh_token_123", ttl).
					Return(uint64(1), nil)
			},
			expectedID:    1,
			expectedError: nil,
		},
		{
			name:         "user not found",
			userID:       999,
			refreshToken: "token",
			ttl:          ttl,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					CreateRefreshToken(gomock.Any(), uint64(999), "token", ttl).
					Return(uint64(0), errors.New("user not found"))
			},
			expectedID:    0,
			expectedError: errors.New("user not found"),
		},
		{
			name:         "token already exists for user",
			userID:       1,
			refreshToken: "existing_token",
			ttl:          ttl,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					CreateRefreshToken(gomock.Any(), uint64(1), "existing_token", ttl).
					Return(uint64(0), errors.New("refresh token already exists for user"))
			},
			expectedID:    0,
			expectedError: errors.New("refresh token already exists for user"),
		},
		{
			name:         "database error",
			userID:       1,
			refreshToken: "token",
			ttl:          ttl,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					CreateRefreshToken(gomock.Any(), uint64(1), "token", ttl).
					Return(uint64(0), errors.New("database connection failed"))
			},
			expectedID:    0,
			expectedError: errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockAuthRepository(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := auth.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			id, err := decorator.CreateRefreshToken(ctx, tt.userID, tt.refreshToken, tt.ttl)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedID, id)
		})
	}
}

func TestTraceDecorator_GetRefreshTokenByUserID(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		userID        uint64
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockAuthRepository, *mocktracing.MockSpan)
		expectedToken *domains.RefreshToken
		expectedError error
	}{
		{
			name:   "successful get refresh token with tracing",
			userID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					GetRefreshTokenByUserID(gomock.Any(), uint64(1)).
					Return(&domains.RefreshToken{
						ID:        1,
						UserID:    1,
						Value:     "refresh_token",
						TTL:       now.Add(24 * time.Hour),
						CreatedAt: now,
					}, nil)
			},
			expectedToken: &domains.RefreshToken{
				ID:        1,
				UserID:    1,
				Value:     "refresh_token",
				TTL:       now.Add(24 * time.Hour),
				CreatedAt: now,
			},
			expectedError: nil,
		},
		{
			name:   "refresh token not found",
			userID: 999,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetRefreshTokenByUserID(gomock.Any(), uint64(999)).
					Return(nil, errors.New("refresh token not found"))
			},
			expectedToken: nil,
			expectedError: errors.New("refresh token not found"),
		},
		{
			name:   "expired token",
			userID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetRefreshTokenByUserID(gomock.Any(), uint64(1)).
					Return(&domains.RefreshToken{
						ID:        1,
						UserID:    1,
						Value:     "expired_token",
						TTL:       now.Add(-1 * time.Hour),
						CreatedAt: now.Add(-25 * time.Hour),
					}, nil)
			},
			expectedToken: &domains.RefreshToken{
				ID:        1,
				UserID:    1,
				Value:     "expired_token",
				TTL:       now.Add(-1 * time.Hour),
				CreatedAt: now.Add(-25 * time.Hour),
			},
			expectedError: nil,
		},
		{
			name:   "database error",
			userID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetRefreshTokenByUserID(gomock.Any(), uint64(1)).
					Return(nil, errors.New("database connection failed"))
			},
			expectedToken: nil,
			expectedError: errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockAuthRepository(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := auth.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			token, err := decorator.GetRefreshTokenByUserID(ctx, tt.userID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedToken, token)
		})
	}
}

func TestTraceDecorator_ExpireRefreshToken(t *testing.T) {
	tests := []struct {
		name           string
		refreshTokenID uint64
		setupMocks     func(*mocktracing.MockProvider, *mockrepositories.MockAuthRepository, *mocktracing.MockSpan)
		expectedError  error
	}{
		{
			name:           "successful expire refresh token with tracing",
			refreshTokenID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					ExpireRefreshToken(gomock.Any(), uint64(1)).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:           "token not found",
			refreshTokenID: 999,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ExpireRefreshToken(gomock.Any(), uint64(999)).
					Return(errors.New("refresh token not found"))
			},
			expectedError: errors.New("refresh token not found"),
		},
		{
			name:           "token already expired",
			refreshTokenID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ExpireRefreshToken(gomock.Any(), uint64(1)).
					Return(errors.New("token already expired"))
			},
			expectedError: errors.New("token already expired"),
		},
		{
			name:           "database error",
			refreshTokenID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ExpireRefreshToken(gomock.Any(), uint64(1)).
					Return(errors.New("database connection failed"))
			},
			expectedError: errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockAuthRepository(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := auth.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			err := decorator.ExpireRefreshToken(ctx, tt.refreshTokenID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTraceDecorator_VerifyEmail(t *testing.T) {
	tests := []struct {
		name          string
		userID        uint64
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockAuthRepository, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name:   "successful verify email with tracing",
			userID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					VerifyEmail(gomock.Any(), uint64(1)).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:   "user not found",
			userID: 999,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					VerifyEmail(gomock.Any(), uint64(999)).
					Return(errors.New("user not found"))
			},
			expectedError: errors.New("user not found"),
		},
		{
			name:   "email already verified",
			userID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					VerifyEmail(gomock.Any(), uint64(1)).
					Return(errors.New("email already verified"))
			},
			expectedError: errors.New("email already verified"),
		},
		{
			name:   "database error",
			userID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					VerifyEmail(gomock.Any(), uint64(1)).
					Return(errors.New("database connection failed"))
			},
			expectedError: errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockAuthRepository(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := auth.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			err := decorator.VerifyEmail(ctx, tt.userID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTraceDecorator_ChangePassword(t *testing.T) {
	tests := []struct {
		name          string
		userID        uint64
		newPassword   string
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockAuthRepository, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name:        "successful change password with tracing",
			userID:      1,
			newPassword: "NewPassword123!",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					ChangePassword(gomock.Any(), uint64(1), "NewPassword123!").
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:        "user not found",
			userID:      999,
			newPassword: "NewPassword123!",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ChangePassword(gomock.Any(), uint64(999), "NewPassword123!").
					Return(errors.New("user not found"))
			},
			expectedError: errors.New("user not found"),
		},
		{
			name:        "weak password",
			userID:      1,
			newPassword: "weak",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ChangePassword(gomock.Any(), uint64(1), "weak").
					Return(errors.New("password does not meet security requirements"))
			},
			expectedError: errors.New("password does not meet security requirements"),
		},
		{
			name:        "same as old password",
			userID:      1,
			newPassword: "SamePassword123!",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ChangePassword(gomock.Any(), uint64(1), "SamePassword123!").
					Return(errors.New("new password must be different from old password"))
			},
			expectedError: errors.New("new password must be different from old password"),
		},
		{
			name:        "database error",
			userID:      1,
			newPassword: "NewPassword123!",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockAuthRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ChangePassword(gomock.Any(), uint64(1), "NewPassword123!").
					Return(errors.New("database connection failed"))
			},
			expectedError: errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockAuthRepository(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := auth.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			err := decorator.ChangePassword(ctx, tt.userID, tt.newPassword)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
