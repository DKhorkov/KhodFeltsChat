package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	internalerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/usecases/auth"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/cache"
	mockcache "github.com/DKhorkov/libs/cache/mocks"
	"github.com/DKhorkov/libs/logging"
	"github.com/DKhorkov/libs/logging/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewCacheDecorator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		cacheProvider cache.Provider
		logger        logging.Logger
		base          interfaces.AuthUseCases
	}{
		{
			name:          "create cache decorator with valid params",
			cacheProvider: mockcache.NewMockProvider(gomock.NewController(t)),
			logger:        mocks.NewMockLogger(gomock.NewController(t)),
			base:          mockusecases.NewMockAuthUseCases(gomock.NewController(t)),
		},
		{
			name:          "create cache decorator with nil base",
			cacheProvider: mockcache.NewMockProvider(gomock.NewController(t)),
			logger:        mocks.NewMockLogger(gomock.NewController(t)),
			base:          nil,
		},
		{
			name:          "create cache decorator with nil cache provider",
			cacheProvider: nil,
			logger:        mocks.NewMockLogger(gomock.NewController(t)),
			base:          mockusecases.NewMockAuthUseCases(gomock.NewController(t)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decorator := auth.NewCacheDecorator(tt.cacheProvider, tt.logger, tt.base)

			assert.NotNil(t, decorator)
		})
	}
}

func TestCacheDecorator_SendVerifyEmailMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		email         string
		setupMocks    func(*mockcache.MockProvider, *mocks.MockLogger, *mockusecases.MockAuthUseCases)
		expectedError error
	}{
		{
			name:  "successful send with cache miss - set initial counter",
			email: "user@example.com",
			setupMocks: func(
				mockCache *mockcache.MockProvider,
				mockLogger *mocks.MockLogger,
				mockBase *mockusecases.MockAuthUseCases,
			) {
				mockCache.EXPECT().
					Ping(gomock.Any()).
					Return("PONG", nil)

				mockCache.EXPECT().
					Get(gomock.Any(), "email_verification:user@example.com").
					Return("", errors.New("key not found"))

				mockLogger.EXPECT().
					ErrorContext(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)

				mockBase.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), "user@example.com").
					Return(nil)

				mockCache.EXPECT().
					Set(gomock.Any(), "email_verification:user@example.com", 1, time.Minute).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:  "successful send with existing counter - increment",
			email: "user@example.com",
			setupMocks: func(
				mockCache *mockcache.MockProvider,
				_ *mocks.MockLogger,
				mockBase *mockusecases.MockAuthUseCases,
			) {
				mockCache.EXPECT().
					Ping(gomock.Any()).
					Return("PONG", nil)

				mockCache.EXPECT().
					Get(gomock.Any(), "email_verification:user@example.com").
					Return("2", nil)

				mockBase.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), "user@example.com").
					Return(nil)

				mockCache.EXPECT().
					Incr(gomock.Any(), "email_verification:user@example.com").
					Return(int64(3), nil)
			},
			expectedError: nil,
		},
		{
			name:  "rate limit exceeded",
			email: "user@example.com",
			setupMocks: func(
				mockCache *mockcache.MockProvider,
				_ *mocks.MockLogger,
				_ *mockusecases.MockAuthUseCases,
			) {
				mockCache.EXPECT().
					Ping(gomock.Any()).
					Return("PONG", nil)

				mockCache.EXPECT().
					Get(gomock.Any(), "email_verification:user@example.com").
					Return("3", nil)
			},
			expectedError: internalerrors.ErrLimitExceeded,
		},
		{
			name:  "cache ping fails - fallback to base",
			email: "user@example.com",
			setupMocks: func(
				mockCache *mockcache.MockProvider,
				_ *mocks.MockLogger,
				mockBase *mockusecases.MockAuthUseCases,
			) {
				mockCache.EXPECT().
					Ping(gomock.Any()).
					Return("", errors.New("cache unavailable"))

				mockBase.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), "user@example.com").
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:  "base returns error",
			email: "user@example.com",
			setupMocks: func(
				mockCache *mockcache.MockProvider,
				_ *mocks.MockLogger,
				mockBase *mockusecases.MockAuthUseCases,
			) {
				mockCache.EXPECT().
					Ping(gomock.Any()).
					Return("PONG", nil)

				mockCache.EXPECT().
					Get(gomock.Any(), "email_verification:user@example.com").
					Return("", nil)

				mockBase.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), "user@example.com").
					Return(errors.New("user not found"))
			},
			expectedError: errors.New("user not found"),
		},
		{
			name:  "cache get error - continue with base",
			email: "user@example.com",
			setupMocks: func(
				mockCache *mockcache.MockProvider,
				mockLogger *mocks.MockLogger,
				mockBase *mockusecases.MockAuthUseCases,
			) {
				mockCache.EXPECT().
					Ping(gomock.Any()).
					Return("PONG", nil)

				mockCache.EXPECT().
					Get(gomock.Any(), "email_verification:user@example.com").
					Return("", errors.New("redis error"))

				mockLogger.EXPECT().
					ErrorContext(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)

				mockBase.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), "user@example.com").
					Return(nil)

				mockCache.EXPECT().
					Set(gomock.Any(), "email_verification:user@example.com", 1, time.Minute).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:  "invalid counter value in cache",
			email: "user@example.com",
			setupMocks: func(
				mockCache *mockcache.MockProvider,
				mockLogger *mocks.MockLogger,
				mockBase *mockusecases.MockAuthUseCases,
			) {
				mockCache.EXPECT().
					Ping(gomock.Any()).
					Return("PONG", nil)

				mockCache.EXPECT().
					Get(gomock.Any(), "email_verification:user@example.com").
					Return("invalid", nil)

				mockLogger.EXPECT().
					ErrorContext(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)

				mockBase.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), "user@example.com").
					Return(nil)

				mockCache.EXPECT().
					Set(gomock.Any(), "email_verification:user@example.com", 1, time.Minute).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:  "cache set fails - log error but return success",
			email: "user@example.com",
			setupMocks: func(
				mockCache *mockcache.MockProvider,
				mockLogger *mocks.MockLogger,
				mockBase *mockusecases.MockAuthUseCases,
			) {
				mockCache.EXPECT().
					Ping(gomock.Any()).
					Return("PONG", nil)

				mockCache.EXPECT().
					Get(gomock.Any(), "email_verification:user@example.com").
					Return("", nil)

				mockBase.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), "user@example.com").
					Return(nil)

				mockCache.EXPECT().
					Set(gomock.Any(), "email_verification:user@example.com", 1, time.Minute).
					Return(errors.New("set failed"))

				mockLogger.EXPECT().
					ErrorContext(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)
			},
			expectedError: nil,
		},
		{
			name:  "cache increment fails - log error but return success",
			email: "user@example.com",
			setupMocks: func(
				mockCache *mockcache.MockProvider,
				mockLogger *mocks.MockLogger,
				mockBase *mockusecases.MockAuthUseCases,
			) {
				mockCache.EXPECT().
					Ping(gomock.Any()).
					Return("PONG", nil)

				mockCache.EXPECT().
					Get(gomock.Any(), "email_verification:user@example.com").
					Return("2", nil)

				mockBase.EXPECT().
					SendVerifyEmailMessage(gomock.Any(), "user@example.com").
					Return(nil)

				mockCache.EXPECT().
					Incr(gomock.Any(), "email_verification:user@example.com").
					Return(int64(0), errors.New("increment failed"))

				mockLogger.EXPECT().
					ErrorContext(
						gomock.Any(),
						"Failed to increment cache for email_verification:user@example.com key",
						gomock.Any(),
					).
					Times(1)
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockCache := mockcache.NewMockProvider(ctrl)
			mockLogger := mocks.NewMockLogger(ctrl)
			mockBase := mockusecases.NewMockAuthUseCases(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockCache, mockLogger, mockBase)
			}

			decorator := auth.NewCacheDecorator(mockCache, mockLogger, mockBase)

			ctx := context.Background()
			err := decorator.SendVerifyEmailMessage(ctx, tt.email)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCacheDecorator_SendForgetPasswordMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		email         string
		setupMocks    func(*mockcache.MockProvider, *mocks.MockLogger, *mockusecases.MockAuthUseCases)
		expectedError error
	}{
		{
			name:  "successful send with cache miss - set initial counter",
			email: "user@example.com",
			setupMocks: func(
				mockCache *mockcache.MockProvider,
				mockLogger *mocks.MockLogger,
				mockBase *mockusecases.MockAuthUseCases,
			) {
				mockCache.EXPECT().
					Ping(gomock.Any()).
					Return("PONG", nil)

				mockCache.EXPECT().
					Get(gomock.Any(), "forget_password:user@example.com").
					Return("", errors.New("key not found"))

				mockLogger.EXPECT().
					ErrorContext(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)

				mockBase.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@example.com").
					Return(nil)

				mockCache.EXPECT().
					Set(gomock.Any(), "forget_password:user@example.com", 1, time.Minute).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:  "successful send with existing counter - increment",
			email: "user@example.com",
			setupMocks: func(
				mockCache *mockcache.MockProvider,
				_ *mocks.MockLogger,
				mockBase *mockusecases.MockAuthUseCases,
			) {
				mockCache.EXPECT().
					Ping(gomock.Any()).
					Return("PONG", nil)

				mockCache.EXPECT().
					Get(gomock.Any(), "forget_password:user@example.com").
					Return("2", nil)

				mockBase.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@example.com").
					Return(nil)

				mockCache.EXPECT().
					Incr(gomock.Any(), "forget_password:user@example.com").
					Return(int64(3), nil)
			},
			expectedError: nil,
		},
		{
			name:  "rate limit exceeded",
			email: "user@example.com",
			setupMocks: func(
				mockCache *mockcache.MockProvider,
				_ *mocks.MockLogger,
				_ *mockusecases.MockAuthUseCases,
			) {
				mockCache.EXPECT().
					Ping(gomock.Any()).
					Return("PONG", nil)

				mockCache.EXPECT().
					Get(gomock.Any(), "forget_password:user@example.com").
					Return("3", nil)
			},
			expectedError: internalerrors.ErrLimitExceeded,
		},
		{
			name:  "cache ping fails - fallback to base",
			email: "user@example.com",
			setupMocks: func(
				mockCache *mockcache.MockProvider,
				_ *mocks.MockLogger,
				mockBase *mockusecases.MockAuthUseCases,
			) {
				mockCache.EXPECT().
					Ping(gomock.Any()).
					Return("", errors.New("cache unavailable"))

				mockBase.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@example.com").
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:  "base returns error",
			email: "user@example.com",
			setupMocks: func(
				mockCache *mockcache.MockProvider,
				_ *mocks.MockLogger,
				mockBase *mockusecases.MockAuthUseCases,
			) {
				mockCache.EXPECT().
					Ping(gomock.Any()).
					Return("PONG", nil)

				mockCache.EXPECT().
					Get(gomock.Any(), "forget_password:user@example.com").
					Return("", nil)

				mockBase.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@example.com").
					Return(errors.New("user not found"))
			},
			expectedError: errors.New("user not found"),
		},
		{
			name:  "cache get error - continue with base",
			email: "user@example.com",
			setupMocks: func(
				mockCache *mockcache.MockProvider,
				mockLogger *mocks.MockLogger,
				mockBase *mockusecases.MockAuthUseCases,
			) {
				mockCache.EXPECT().
					Ping(gomock.Any()).
					Return("PONG", nil)

				mockCache.EXPECT().
					Get(gomock.Any(), "forget_password:user@example.com").
					Return("", errors.New("redis error"))

				mockLogger.EXPECT().
					ErrorContext(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)

				mockBase.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@example.com").
					Return(nil)

				mockCache.EXPECT().
					Set(gomock.Any(), "forget_password:user@example.com", 1, time.Minute).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:  "invalid counter value in cache",
			email: "user@example.com",
			setupMocks: func(
				mockCache *mockcache.MockProvider,
				mockLogger *mocks.MockLogger,
				mockBase *mockusecases.MockAuthUseCases,
			) {
				mockCache.EXPECT().
					Ping(gomock.Any()).
					Return("PONG", nil)

				mockCache.EXPECT().
					Get(gomock.Any(), "forget_password:user@example.com").
					Return("invalid", nil)

				mockLogger.EXPECT().
					ErrorContext(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)

				mockBase.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@example.com").
					Return(nil)

				mockCache.EXPECT().
					Set(gomock.Any(), "forget_password:user@example.com", 1, time.Minute).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:  "cache set fails - log error but return success",
			email: "user@example.com",
			setupMocks: func(
				mockCache *mockcache.MockProvider,
				mockLogger *mocks.MockLogger,
				mockBase *mockusecases.MockAuthUseCases,
			) {
				mockCache.EXPECT().
					Ping(gomock.Any()).
					Return("PONG", nil)

				mockCache.EXPECT().
					Get(gomock.Any(), "forget_password:user@example.com").
					Return("", nil)

				mockBase.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@example.com").
					Return(nil)

				mockCache.EXPECT().
					Set(gomock.Any(), "forget_password:user@example.com", 1, time.Minute).
					Return(errors.New("set failed"))

				mockLogger.EXPECT().
					ErrorContext(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1)
			},
			expectedError: nil,
		},
		{
			name:  "cache increment fails - log error but return success",
			email: "user@example.com",
			setupMocks: func(
				mockCache *mockcache.MockProvider,
				mockLogger *mocks.MockLogger,
				mockBase *mockusecases.MockAuthUseCases,
			) {
				mockCache.EXPECT().
					Ping(gomock.Any()).
					Return("PONG", nil)

				mockCache.EXPECT().
					Get(gomock.Any(), "forget_password:user@example.com").
					Return("2", nil)

				mockBase.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@example.com").
					Return(nil)

				mockCache.EXPECT().
					Incr(gomock.Any(), "forget_password:user@example.com").
					Return(int64(0), errors.New("increment failed"))

				mockLogger.EXPECT().
					ErrorContext(
						gomock.Any(),
						"Failed to increment cache for forget_password:user@example.com key",
						gomock.Any(),
					).
					Times(1)
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockCache := mockcache.NewMockProvider(ctrl)
			mockLogger := mocks.NewMockLogger(ctrl)
			mockBase := mockusecases.NewMockAuthUseCases(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockCache, mockLogger, mockBase)
			}

			decorator := auth.NewCacheDecorator(mockCache, mockLogger, mockBase)

			ctx := context.Background()
			err := decorator.SendForgetPasswordMessage(ctx, tt.email)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Тест для методов, которые просто пробрасывают вызовы.
func TestCacheDecorator_PassthroughMethods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupMocks    func(*mockusecases.MockAuthUseCases)
		testFunc      func(*auth.CacheDecorator, context.Context) error
		expectedError error
	}{
		{
			name: "RegisterUser",
			setupMocks: func(mockBase *mockusecases.MockAuthUseCases) {
				mockBase.EXPECT().
					RegisterUser(gomock.Any(), gomock.Any()).
					Return(&domains.User{ID: 1}, nil)
			},
			testFunc: func(d *auth.CacheDecorator, ctx context.Context) error {
				_, err := d.RegisterUser(ctx, domains.RegisterDTO{})

				return err
			},
			expectedError: nil,
		},
		{
			name: "LoginUser",
			setupMocks: func(mockBase *mockusecases.MockAuthUseCases) {
				mockBase.EXPECT().
					LoginUser(gomock.Any(), gomock.Any()).
					Return(&domains.TokensDTO{AccessToken: "token"}, nil)
			},
			testFunc: func(d *auth.CacheDecorator, ctx context.Context) error {
				_, err := d.LoginUser(ctx, domains.LoginDTO{})

				return err
			},
			expectedError: nil,
		},
		{
			name: "RefreshTokens",
			setupMocks: func(mockBase *mockusecases.MockAuthUseCases) {
				mockBase.EXPECT().
					RefreshTokens(gomock.Any(), "token").
					Return(&domains.TokensDTO{AccessToken: "new_token"}, nil)
			},
			testFunc: func(d *auth.CacheDecorator, ctx context.Context) error {
				_, err := d.RefreshTokens(ctx, "token")

				return err
			},
			expectedError: nil,
		},
		{
			name: "LogoutUser",
			setupMocks: func(mockBase *mockusecases.MockAuthUseCases) {
				mockBase.EXPECT().
					LogoutUser(gomock.Any(), uint64(1)).
					Return(nil)
			},
			testFunc: func(d *auth.CacheDecorator, ctx context.Context) error {
				return d.LogoutUser(ctx, 1)
			},
			expectedError: nil,
		},
		{
			name: "VerifyEmail",
			setupMocks: func(mockBase *mockusecases.MockAuthUseCases) {
				mockBase.EXPECT().
					VerifyEmail(gomock.Any(), "token").
					Return(nil)
			},
			testFunc: func(d *auth.CacheDecorator, ctx context.Context) error {
				return d.VerifyEmail(ctx, "token")
			},
			expectedError: nil,
		},
		{
			name: "ForgetPassword",
			setupMocks: func(mockBase *mockusecases.MockAuthUseCases) {
				mockBase.EXPECT().
					ForgetPassword(gomock.Any(), "token", "newpass").
					Return(nil)
			},
			testFunc: func(d *auth.CacheDecorator, ctx context.Context) error {
				return d.ForgetPassword(ctx, "token", "newpass")
			},
			expectedError: nil,
		},
		{
			name: "ChangePassword",
			setupMocks: func(mockBase *mockusecases.MockAuthUseCases) {
				mockBase.EXPECT().
					ChangePassword(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			testFunc: func(d *auth.CacheDecorator, ctx context.Context) error {
				return d.ChangePassword(ctx, domains.ChangePasswordDTO{})
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockCache := mockcache.NewMockProvider(ctrl)
			mockLogger := mocks.NewMockLogger(ctrl)
			mockBase := mockusecases.NewMockAuthUseCases(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockBase)
			}

			decorator := auth.NewCacheDecorator(mockCache, mockLogger, mockBase)

			ctx := context.Background()
			err := tt.testFunc(decorator, ctx)

			if tt.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
