package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/usecases/auth"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
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
		base          interfaces.AuthUseCases
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
			base: mockusecases.NewMockAuthUseCases(gomock.NewController(t)),
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
			base:          mockusecases.NewMockAuthUseCases(gomock.NewController(t)),
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
		dto           domains.RegisterDTO
		setupMocks    func(*mocktracing.MockProvider, *mockusecases.MockAuthUseCases, *mocktracing.MockSpan)
		expectedUser  *domains.User
		expectedError error
	}{
		{
			name: "successful registration with tracing",
			dto: domains.RegisterDTO{
				Username: "newuser",
				Email:    "newuser@example.com",
				Password: "Password123!",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
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
			dto: domains.RegisterDTO{
				Username: "existing",
				Email:    "existing@example.com",
				Password: "Password123!",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
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
			dto: domains.RegisterDTO{
				Username: "weakuser",
				Email:    "weak@example.com",
				Password: "weak",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
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
			mockBase := mockusecases.NewMockAuthUseCases(ctrl)
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
			user, err := decorator.RegisterUser(ctx, tt.dto)

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

func TestTraceDecorator_LoginUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		dto            domains.LoginDTO
		setupMocks     func(*mocktracing.MockProvider, *mockusecases.MockAuthUseCases, *mocktracing.MockSpan)
		expectedTokens *domains.TokensDTO
		expectedError  error
	}{
		{
			name: "successful login with tracing",
			dto: domains.LoginDTO{
				Login:    "user@example.com",
				Password: "Password123!",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					LoginUser(gomock.Any(), domains.LoginDTO{
						Login:    "user@example.com",
						Password: "Password123!",
					}).
					Return(&domains.TokensDTO{
						AccessToken:  "access_token_123",
						RefreshToken: "refresh_token_456",
					}, nil)
			},
			expectedTokens: &domains.TokensDTO{
				AccessToken:  "access_token_123",
				RefreshToken: "refresh_token_456",
			},
			expectedError: nil,
		},
		{
			name: "login with invalid credentials",
			dto: domains.LoginDTO{
				Login:    "wrong@example.com",
				Password: "wrong",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					LoginUser(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("invalid credentials"))
			},
			expectedTokens: nil,
			expectedError:  errors.New("invalid credentials"),
		},
		{
			name: "user not found",
			dto: domains.LoginDTO{
				Login:    "nonexistent@example.com",
				Password: "Password123!",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					LoginUser(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("user not found"))
			},
			expectedTokens: nil,
			expectedError:  errors.New("user not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockusecases.NewMockAuthUseCases(ctrl)
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
			tokens, err := decorator.LoginUser(ctx, tt.dto)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedTokens, tokens)
		})
	}
}

func TestTraceDecorator_RefreshTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		refreshToken   string
		setupMocks     func(*mocktracing.MockProvider, *mockusecases.MockAuthUseCases, *mocktracing.MockSpan)
		expectedTokens *domains.TokensDTO
		expectedError  error
	}{
		{
			name:         "successful refresh tokens with tracing",
			refreshToken: "valid_refresh_token",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					RefreshTokens(gomock.Any(), "valid_refresh_token").
					Return(&domains.TokensDTO{
						AccessToken:  "new_access_token",
						RefreshToken: "new_refresh_token",
					}, nil)
			},
			expectedTokens: &domains.TokensDTO{
				AccessToken:  "new_access_token",
				RefreshToken: "new_refresh_token",
			},
			expectedError: nil,
		},
		{
			name:         "invalid refresh token",
			refreshToken: "invalid_token",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					RefreshTokens(gomock.Any(), "invalid_token").
					Return(nil, errors.New("invalid refresh token"))
			},
			expectedTokens: nil,
			expectedError:  errors.New("invalid refresh token"),
		},
		{
			name:         "expired refresh token",
			refreshToken: "expired_token",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					RefreshTokens(gomock.Any(), "expired_token").
					Return(nil, errors.New("refresh token expired"))
			},
			expectedTokens: nil,
			expectedError:  errors.New("refresh token expired"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockusecases.NewMockAuthUseCases(ctrl)
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
			tokens, err := decorator.RefreshTokens(ctx, tt.refreshToken)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedTokens, tokens)
		})
	}
}

func TestTraceDecorator_LogoutUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		refreshToken  string
		setupMocks    func(*mocktracing.MockProvider, *mockusecases.MockAuthUseCases, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name:         "successful logout with tracing",
			refreshToken: "valid_refresh_token",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					LogoutUser(gomock.Any(), "valid_refresh_token").
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:         "logout error",
			refreshToken: "valid_refresh_token",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					LogoutUser(gomock.Any(), "valid_refresh_token").
					Return(errors.New("logout error"))
			},
			expectedError: errors.New("logout error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockusecases.NewMockAuthUseCases(ctrl)
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
			err := decorator.LogoutUser(ctx, tt.refreshToken)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTraceDecorator_LogoutUserFromAllSessions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		userID        uint64
		setupMocks    func(*mocktracing.MockProvider, *mockusecases.MockAuthUseCases, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name:   "successful logout from all sessions with tracing",
			userID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					LogoutUserFromAllSessions(gomock.Any(), uint64(1)).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:   "database error",
			userID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					LogoutUserFromAllSessions(gomock.Any(), uint64(1)).
					Return(errors.New("database error"))
			},
			expectedError: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockusecases.NewMockAuthUseCases(ctrl)
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
			err := decorator.LogoutUserFromAllSessions(ctx, tt.userID)

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
		name            string
		verifyEmailCode uint64
		setupMocks      func(*mocktracing.MockProvider, *mockusecases.MockAuthUseCases, *mocktracing.MockSpan)
		expectedError   error
	}{
		{
			name:            "successful email verification with tracing",
			verifyEmailCode: 111111,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					VerifyEmail(gomock.Any(), uint64(111111)).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:            "invalid code",
			verifyEmailCode: 222222,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					VerifyEmail(gomock.Any(), uint64(222222)).
					Return(errors.New("invalid verification token"))
			},
			expectedError: errors.New("invalid verification token"),
		},
		{
			name:            "expired code",
			verifyEmailCode: 333333,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					VerifyEmail(gomock.Any(), uint64(333333)).
					Return(errors.New("verification token expired"))
			},
			expectedError: errors.New("verification token expired"),
		},
		{
			name:            "email already verified",
			verifyEmailCode: 444444,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					VerifyEmail(gomock.Any(), uint64(444444)).
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
			mockBase := mockusecases.NewMockAuthUseCases(ctrl)
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
			err := decorator.VerifyEmail(ctx, tt.verifyEmailCode)

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
		name               string
		forgetPasswordCode uint64
		newPassword        string
		setupMocks         func(*mocktracing.MockProvider, *mockusecases.MockAuthUseCases, *mocktracing.MockSpan)
		expectedError      error
	}{
		{
			name:               "successful password reset with tracing",
			forgetPasswordCode: 111111,
			newPassword:        "NewPassword123!",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					ForgetPassword(gomock.Any(), uint64(111111), "NewPassword123!").
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:               "invalid code",
			forgetPasswordCode: 222222,
			newPassword:        "NewPassword123!",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ForgetPassword(gomock.Any(), uint64(222222), "NewPassword123!").
					Return(errors.New("invalid reset token"))
			},
			expectedError: errors.New("invalid reset token"),
		},
		{
			name:               "expired code",
			forgetPasswordCode: 333333,
			newPassword:        "NewPassword123!",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ForgetPassword(gomock.Any(), uint64(333333), "NewPassword123!").
					Return(errors.New("reset token expired"))
			},
			expectedError: errors.New("reset token expired"),
		},
		{
			name:               "weak password",
			forgetPasswordCode: 444444,
			newPassword:        "weak",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ForgetPassword(gomock.Any(), uint64(444444), "weak").
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
			mockBase := mockusecases.NewMockAuthUseCases(ctrl)
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
			err := decorator.ForgetPassword(ctx, tt.forgetPasswordCode, tt.newPassword)

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
		dto           domains.ChangePasswordDTO
		setupMocks    func(*mocktracing.MockProvider, *mockusecases.MockAuthUseCases, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name: "successful password change with tracing",
			dto: domains.ChangePasswordDTO{
				UserID:      1,
				OldPassword: "OldPassword123!",
				NewPassword: "NewPassword123!",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					ChangePassword(gomock.Any(), domains.ChangePasswordDTO{
						UserID:      1,
						OldPassword: "OldPassword123!",
						NewPassword: "NewPassword123!",
					}).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "incorrect old password",
			dto: domains.ChangePasswordDTO{
				UserID:      1,
				OldPassword: "WrongPassword",
				NewPassword: "NewPassword123!",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ChangePassword(gomock.Any(), gomock.Any()).
					Return(errors.New("incorrect old password"))
			},
			expectedError: errors.New("incorrect old password"),
		},
		{
			name: "weak new password",
			dto: domains.ChangePasswordDTO{
				UserID:      1,
				OldPassword: "OldPassword123!",
				NewPassword: "weak",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ChangePassword(gomock.Any(), gomock.Any()).
					Return(errors.New("password does not meet security requirements"))
			},
			expectedError: errors.New("password does not meet security requirements"),
		},
		{
			name: "same as old password",
			dto: domains.ChangePasswordDTO{
				UserID:      1,
				OldPassword: "Password123!",
				NewPassword: "Password123!",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ChangePassword(gomock.Any(), gomock.Any()).
					Return(errors.New("new password must be different from old password"))
			},
			expectedError: errors.New("new password must be different from old password"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockusecases.NewMockAuthUseCases(ctrl)
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
			err := decorator.ChangePassword(ctx, tt.dto)

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
		setupMocks    func(*mocktracing.MockProvider, *mockusecases.MockAuthUseCases, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name:  "successful send verify email with tracing",
			email: "user@example.com",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
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
				mockBase *mockusecases.MockAuthUseCases,
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
				mockBase *mockusecases.MockAuthUseCases,
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
				mockBase *mockusecases.MockAuthUseCases,
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
				mockBase *mockusecases.MockAuthUseCases,
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
			mockBase := mockusecases.NewMockAuthUseCases(ctrl)
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

func TestTraceDecorator_SendForgetPasswordMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		email         string
		setupMocks    func(*mocktracing.MockProvider, *mockusecases.MockAuthUseCases, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name:  "successful send forget password message with tracing",
			email: "user@example.com",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
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
				mockBase *mockusecases.MockAuthUseCases,
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
				mockBase *mockusecases.MockAuthUseCases,
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
				mockBase *mockusecases.MockAuthUseCases,
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
		{
			name:  "empty email",
			email: "",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockAuthUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "").
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
			mockBase := mockusecases.NewMockAuthUseCases(ctrl)
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
