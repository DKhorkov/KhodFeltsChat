package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/services/auth"
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
		base          interfaces.AuthService
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
			base: mockservices.NewMockAuthService(gomock.NewController(t)),
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
			base:          mockservices.NewMockAuthService(gomock.NewController(t)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decorator := auth.NewTraceDecorator(tt.traceProvider, tt.spanConfig, tt.base)

			assert.NotNil(t, decorator)
		})
	}
}

func TestTraceDecorator_RegisterUser(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		userData      domains.RegisterDTO
		setupMocks    func(*mocktracing.MockProvider, *mockservices.MockAuthService, *mocktracing.MockSpan)
		expectedUser  *domains.User
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
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					RegisterUser(gomock.Any(), domains.RegisterDTO{
						Username: "newuser",
						Email:    "newuser@example.com",
						Password: "Password123!",
					}).
					Return(&domains.User{
						ID:        1,
						Username:  "newuser",
						Email:     "newuser@example.com",
						CreatedAt: now,
						UpdatedAt: now,
					}, nil)
			},
			expectedUser: &domains.User{
				ID:        1,
				Username:  "newuser",
				Email:     "newuser@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
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
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					RegisterUser(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("email already exists"))
			},
			expectedUser:  nil,
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
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					RegisterUser(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("password does not meet security requirements"))
			},
			expectedUser:  nil,
			expectedError: errors.New("password does not meet security requirements"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockAuthService(ctrl)
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
			user, err := decorator.RegisterUser(ctx, tt.userData)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedUser, user)
		})
	}
}

func TestTraceDecorator_CreateRefreshToken(t *testing.T) {
	t.Parallel()

	now := time.Now()
	ttl := 24 * time.Hour

	tests := []struct {
		name          string
		userID        uint64
		value         string
		ttl           time.Duration
		setupMocks    func(*mocktracing.MockProvider, *mockservices.MockAuthService, *mocktracing.MockSpan)
		expectedToken *domains.RefreshToken
		expectedError error
	}{
		{
			name:   "successful create refresh token with tracing",
			userID: 1,
			value:  "refresh_token_123",
			ttl:    ttl,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					CreateRefreshToken(gomock.Any(), uint64(1), "refresh_token_123", ttl).
					Return(&domains.RefreshToken{
						ID:        1,
						UserID:    1,
						Value:     "refresh_token_123",
						TTL:       now.Add(ttl),
						CreatedAt: now,
					}, nil)
			},
			expectedToken: &domains.RefreshToken{
				ID:        1,
				UserID:    1,
				Value:     "refresh_token_123",
				TTL:       now.Add(ttl),
				CreatedAt: now,
			},
			expectedError: nil,
		},
		{
			name:   "user not found",
			userID: 999,
			value:  "token",
			ttl:    ttl,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					CreateRefreshToken(gomock.Any(), uint64(999), "token", ttl).
					Return(nil, errors.New("user not found"))
			},
			expectedToken: nil,
			expectedError: errors.New("user not found"),
		},
		{
			name:   "token already exists",
			userID: 1,
			value:  "existing_token",
			ttl:    ttl,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					CreateRefreshToken(gomock.Any(), uint64(1), "existing_token", ttl).
					Return(nil, errors.New("refresh token already exists for user"))
			},
			expectedToken: nil,
			expectedError: errors.New("refresh token already exists for user"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockAuthService(ctrl)
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
			token, err := decorator.CreateRefreshToken(ctx, tt.userID, tt.value, tt.ttl)

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

func TestTraceDecorator_GetRefreshTokenByUserID(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		userID        uint64
		setupMocks    func(*mocktracing.MockProvider, *mockservices.MockAuthService, *mocktracing.MockSpan)
		expectedToken *domains.RefreshToken
		expectedError error
	}{
		{
			name:   "successful get refresh token with tracing",
			userID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
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
				mockBase *mockservices.MockAuthService,
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
				mockBase *mockservices.MockAuthService,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockAuthService(ctrl)
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
	t.Parallel()

	tests := []struct {
		name           string
		refreshTokenID uint64
		setupMocks     func(*mocktracing.MockProvider, *mockservices.MockAuthService, *mocktracing.MockSpan)
		expectedError  error
	}{
		{
			name:           "successful expire refresh token with tracing",
			refreshTokenID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
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
				mockBase *mockservices.MockAuthService,
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
				mockBase *mockservices.MockAuthService,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockAuthService(ctrl)
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
	t.Parallel()

	tests := []struct {
		name          string
		userID        uint64
		setupMocks    func(*mocktracing.MockProvider, *mockservices.MockAuthService, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name:   "successful verify email with tracing",
			userID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
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
				mockBase *mockservices.MockAuthService,
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
				mockBase *mockservices.MockAuthService,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockAuthService(ctrl)
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

func TestTraceDecorator_ForgetPassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		userID        uint64
		newPassword   string
		setupMocks    func(*mocktracing.MockProvider, *mockservices.MockAuthService, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name:        "successful forget password with tracing",
			userID:      1,
			newPassword: "NewPassword123!",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					ForgetPassword(gomock.Any(), uint64(1), "NewPassword123!").
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
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ForgetPassword(gomock.Any(), uint64(999), "NewPassword123!").
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
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ForgetPassword(gomock.Any(), uint64(1), "weak").
					Return(errors.New("password does not meet security requirements"))
			},
			expectedError: errors.New("password does not meet security requirements"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockAuthService(ctrl)
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
			err := decorator.ForgetPassword(ctx, tt.userID, tt.newPassword)

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
	t.Parallel()

	tests := []struct {
		name          string
		userID        uint64
		newPassword   string
		setupMocks    func(*mocktracing.MockProvider, *mockservices.MockAuthService, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name:        "successful change password with tracing",
			userID:      1,
			newPassword: "NewPassword123!",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
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
				mockBase *mockservices.MockAuthService,
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
				mockBase *mockservices.MockAuthService,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockAuthService(ctrl)
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

func TestTraceDecorator_SendForgetPasswordMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		email         string
		setupMocks    func(*mocktracing.MockProvider, *mockservices.MockAuthService, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name:  "successful send forget password message with tracing",
			email: "user@example.com",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@example.com").
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:  "user not found",
			email: "nonexistent@example.com",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "nonexistent@example.com").
					Return(errors.New("user not found"))
			},
			expectedError: errors.New("user not found"),
		},
		{
			name:  "invalid email format",
			email: "invalid-email",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "invalid-email").
					Return(errors.New("invalid email format"))
			},
			expectedError: errors.New("invalid email format"),
		},
		{
			name:  "rate limit exceeded",
			email: "user@example.com",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@example.com").
					Return(errors.New("rate limit exceeded, try again later"))
			},
			expectedError: errors.New("rate limit exceeded, try again later"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockAuthService(ctrl)
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
			err := decorator.SendForgetPasswordMessage(ctx, tt.email)

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

	tests := []struct {
		name          string
		email         string
		setupMocks    func(*mocktracing.MockProvider, *mockservices.MockAuthService, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name:  "successful send verify email message with tracing",
			email: "user@example.com",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), "user@example.com").
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:  "user not found",
			email: "nonexistent@example.com",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), "nonexistent@example.com").
					Return(errors.New("user not found"))
			},
			expectedError: errors.New("user not found"),
		},
		{
			name:  "email already verified",
			email: "verified@example.com",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), "verified@example.com").
					Return(errors.New("email already verified"))
			},
			expectedError: errors.New("email already verified"),
		},
		{
			name:  "invalid email format",
			email: "invalid-email",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), "invalid-email").
					Return(errors.New("invalid email format"))
			},
			expectedError: errors.New("invalid email format"),
		},
		{
			name:  "rate limit exceeded",
			email: "user@example.com",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockAuthService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), "user@example.com").
					Return(errors.New("rate limit exceeded, try again later"))
			},
			expectedError: errors.New("rate limit exceeded, try again later"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockAuthService(ctrl)
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
			err := decorator.SendVerifyEmailMessage(ctx, tt.email)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
