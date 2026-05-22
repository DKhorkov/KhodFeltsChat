package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/common"
	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/usecases/auth"
	mockservices "github.com/DKhorkov/kfc/mocks/services"
	"github.com/DKhorkov/libs/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	hashCost = 5
)

func TestUseCases_RegisterUser(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockAuthService  func(*mockservices.MockAuthService)
		mockUsersService func(*mockservices.MockUsersService)
	}

	type args struct {
		ctx context.Context
		dto domains.RegisterDTO
	}

	pass := "SomeTestP@ssword228"
	hashedPassword, err := security.Hash(pass, hashCost)
	require.NoError(t, err)

	expectedUser := &domains.User{
		ID:       1,
		Username: "testuser",
		Email:    "sometest@gmail.com",
		Password: hashedPassword,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domains.User
		wantErr bool
		err     error
	}{
		{
			name: "successfully register user",
			fields: fields{
				mockAuthService: func(as *mockservices.MockAuthService) {
					as.EXPECT().
						RegisterUser(gomock.Any(), gomock.AssignableToTypeOf(domains.RegisterDTO{})).
						Return(expectedUser, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				dto: domains.RegisterDTO{
					Username: "testuser",
					Email:    "sometest@gmail.com",
					Password: "Password123!",
				},
			},
			want:    expectedUser,
			wantErr: false,
		},
		{
			name: "invalid email format",
			args: args{
				ctx: context.Background(),
				dto: domains.RegisterDTO{
					Username: "testuser",
					Email:    "invalid-email",
					Password: "Password123!",
				},
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrValidationFailed,
		},
		{
			name: "invalid password format",
			args: args{
				ctx: context.Background(),
				dto: domains.RegisterDTO{
					Username: "testuser",
					Email:    "sometest@gmail.com",
					Password: "weak",
				},
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrValidationFailed,
		},
		{
			name: "invalid username format",
			args: args{
				ctx: context.Background(),
				dto: domains.RegisterDTO{
					Username: "ab", // too short
					Email:    "sometest@gmail.com",
					Password: "Password123!",
				},
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrValidationFailed,
		},
		{
			name: "auth service returns error",
			fields: fields{
				mockAuthService: func(as *mockservices.MockAuthService) {
					as.EXPECT().
						RegisterUser(gomock.Any(), gomock.Any()).
						Return(nil, errors.New("user already exists"))
				},
			},
			args: args{
				ctx: context.Background(),
				dto: domains.RegisterDTO{
					Username: "testuser",
					Email:    "sometest@gmail.com",
					Password: "Password123!",
				},
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("user already exists"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockAuthService := mockservices.NewMockAuthService(ctrl)
			mockUsersService := mockservices.NewMockUsersService(ctrl)

			if tt.fields.mockAuthService != nil {
				tt.fields.mockAuthService(mockAuthService)
			}

			if tt.fields.mockUsersService != nil {
				tt.fields.mockUsersService(mockUsersService)
			}

			securityConfig := security.Config{
				HashCost: hashCost,
				JWT: security.JWTConfig{
					SecretKey:       "test-secret",
					AccessTokenTTL:  time.Hour,
					RefreshTokenTTL: time.Hour * 24 * 7,
					Algorithm:       "HS256",
				},
			}

			validationConfig := config.ValidationConfig{
				EmailRegExp: "^[a-z0-9._%+\\-]+@[a-z0-9.\\-]+\\.[a-z]{2,4}$",
				PasswordRegExps: []string{
					".{8,}",
					"[a-z]",
					"[A-Z]",
					"[0-9]",
					"[^\\d\\w]",
				},
				UsernameRegExps: []string{
					`^.{5,70}$`,      // длина 5-70 символов
					`^[A-Za-z0-9]+$`, // только латинница и цифры
				},
			}

			uc := auth.New(
				mockAuthService,
				mockUsersService,
				securityConfig,
				validationConfig,
			)

			// Act
			got, err := uc.RegisterUser(tt.args.ctx, tt.args.dto)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					if errors.Is(tt.err, customerrors.ErrValidationFailed) {
						assert.ErrorIs(t, err, tt.err)
					} else {
						assert.Contains(t, err.Error(), tt.err.Error())
					}
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestUseCases_LoginUser(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockAuthService  func(*mockservices.MockAuthService)
		mockUsersService func(*mockservices.MockUsersService)
	}

	type args struct {
		ctx context.Context
		dto domains.LoginDTO
	}

	pass := "SomeTestP@ssword22"
	hashedPassword, err := security.Hash(pass, hashCost)
	require.NoError(t, err)

	user := &domains.User{
		ID:             123,
		Username:       "testuser",
		Email:          "sometest@gmail.com",
		EmailConfirmed: true,
		Password:       hashedPassword,
	}

	tests := []struct {
		name      string
		fields    fields
		args      args
		wantToken bool
		wantErr   bool
		err       error
	}{
		{
			name: "successfully login by email",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByEmail(gomock.Any(), "sometest@gmail.com").
						Return(user, nil)
				},
				mockAuthService: func(as *mockservices.MockAuthService) {
					as.EXPECT().
						CreateRefreshToken(
							gomock.Any(),
							uint64(123),
							gomock.Any(),
							gomock.Any(),
						).
						Return(&domains.RefreshToken{}, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				dto: domains.LoginDTO{
					Login:    "sometest@gmail.com",
					Password: pass,
				},
			},
			wantToken: true,
			wantErr:   false,
		},
		{
			name: "successfully login by username",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByEmail(gomock.Any(), "testuser").
						Return(nil, errors.New("not found"))
					us.EXPECT().
						GetUserByUsername(gomock.Any(), "testuser").
						Return(user, nil)
				},
				mockAuthService: func(as *mockservices.MockAuthService) {
					as.EXPECT().
						CreateRefreshToken(
							gomock.Any(),
							uint64(123),
							gomock.Any(),
							gomock.Any(),
						).
						Return(&domains.RefreshToken{}, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				dto: domains.LoginDTO{
					Login:    "testuser",
					Password: pass,
				},
			},
			wantToken: true,
			wantErr:   false,
		},
		{
			name: "empty login",
			args: args{
				ctx: context.Background(),
				dto: domains.LoginDTO{
					Login:    "",
					Password: pass,
				},
			},
			wantToken: false,
			wantErr:   true,
			err:       customerrors.ErrValidationFailed,
		},
		{
			name: "empty password",
			args: args{
				ctx: context.Background(),
				dto: domains.LoginDTO{
					Login:    "sometest@gmail.com",
					Password: "",
				},
			},
			wantToken: false,
			wantErr:   true,
			err:       customerrors.ErrValidationFailed,
		},
		{
			name: "user not found by email and username",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByEmail(gomock.Any(), "nonexistent").
						Return(nil, errors.New("not found"))
					us.EXPECT().
						GetUserByUsername(gomock.Any(), "nonexistent").
						Return(nil, errors.New("not found"))
				},
			},
			args: args{
				ctx: context.Background(),
				dto: domains.LoginDTO{
					Login:    "nonexistent",
					Password: pass,
				},
			},
			wantToken: false,
			wantErr:   true,
			err:       customerrors.ErrUserNotFound,
		},
		{
			name: "email not confirmed",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					unconfirmedUser := *user
					unconfirmedUser.EmailConfirmed = false
					us.EXPECT().
						GetUserByEmail(gomock.Any(), "sometest@gmail.com").
						Return(&unconfirmedUser, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				dto: domains.LoginDTO{
					Login:    "sometest@gmail.com",
					Password: pass,
				},
			},
			wantToken: false,
			wantErr:   true,
			err:       customerrors.ErrEmailNotConfirmed,
		},
		{
			name: "error creating refresh token",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByEmail(gomock.Any(), "sometest@gmail.com").
						Return(user, nil)
				},
				mockAuthService: func(as *mockservices.MockAuthService) {
					as.EXPECT().
						CreateRefreshToken(
							gomock.Any(),
							uint64(123),
							gomock.Any(),
							gomock.Any(),
						).
						Return(nil, errors.New("database error"))
				},
			},
			args: args{
				ctx: context.Background(),
				dto: domains.LoginDTO{
					Login:    "sometest@gmail.com",
					Password: pass,
				},
			},
			wantToken: false,
			wantErr:   true,
			err:       errors.New("database error"),
		},
		{
			name: "wrong password",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByEmail(gomock.Any(), "sometest@gmail.com").
						Return(user, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				dto: domains.LoginDTO{
					Login:    "sometest@gmail.com",
					Password: "wrongpassword",
				},
			},
			wantToken: false,
			wantErr:   true,
			err:       customerrors.ErrWrongPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockAuthService := mockservices.NewMockAuthService(ctrl)
			mockUsersService := mockservices.NewMockUsersService(ctrl)

			if tt.fields.mockAuthService != nil {
				tt.fields.mockAuthService(mockAuthService)
			}

			if tt.fields.mockUsersService != nil {
				tt.fields.mockUsersService(mockUsersService)
			}

			securityConfig := security.Config{
				HashCost: hashCost,
				JWT: security.JWTConfig{
					SecretKey:       "test-secret",
					AccessTokenTTL:  time.Hour,
					RefreshTokenTTL: time.Hour * 24 * 7,
					Algorithm:       "HS256",
				},
			}

			validationConfig := config.ValidationConfig{}

			uc := auth.New(
				mockAuthService,
				mockUsersService,
				securityConfig,
				validationConfig,
			)

			// Act
			got, err := uc.LoginUser(tt.args.ctx, tt.args.dto)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					if errors.Is(tt.err, customerrors.ErrValidationFailed) ||
						errors.Is(tt.err, customerrors.ErrEmailNotConfirmed) ||
						errors.Is(tt.err, customerrors.ErrWrongPassword) ||
						errors.Is(tt.err, customerrors.ErrUserNotFound) {
						assert.ErrorIs(t, err, tt.err)
					} else {
						assert.Contains(t, err.Error(), tt.err.Error())
					}
				}
			} else {
				assert.NoError(t, err)
			}

			if tt.wantToken {
				assert.NotEmpty(t, got)
			} else {
				assert.Empty(t, got)
			}
		})
	}
}

func TestUseCases_RefreshTokens(t *testing.T) {
	t.Parallel()

	securityConfig := security.Config{
		JWT: security.JWTConfig{
			SecretKey:       "secret",
			Algorithm:       "HS256",
			AccessTokenTTL:  time.Hour,
			RefreshTokenTTL: time.Hour,
		},
		HashCost: 10,
	}

	validationConfig := config.ValidationConfig{}

	refreshToken, err := security.GenerateJWT(
		uint64(1),
		securityConfig.JWT.SecretKey,
		securityConfig.JWT.RefreshTokenTTL,
		securityConfig.JWT.Algorithm,
	)
	require.NoError(t, err)

	encodedRefreshToken := security.RawEncode([]byte(refreshToken))

	testCases := []struct {
		name         string
		refreshToken string
		setupMocks   func(
			authService *mockservices.MockAuthService,
			usersService *mockservices.MockUsersService,
		)
		expectedErr error
	}{
		{
			name:         "success",
			refreshToken: encodedRefreshToken,
			setupMocks: func(
				authService *mockservices.MockAuthService,
				_ *mockservices.MockUsersService,
			) {
				authService.
					EXPECT().
					GetRefreshTokenByValue(gomock.Any(), refreshToken).
					Return(&domains.RefreshToken{ID: 1, UserID: 1, Value: refreshToken}, nil).
					Times(1)

				authService.
					EXPECT().
					ExpireRefreshToken(gomock.Any(), uint64(1)).
					Return(nil).
					Times(1)

				authService.
					EXPECT().
					CreateRefreshToken(
						gomock.Any(),
						uint64(1),
						gomock.Any(),
						time.Hour,
					).
					Return(&domains.RefreshToken{Value: refreshToken}, nil).
					Times(1)
			},
			expectedErr: nil,
		},
		{
			name:         "invalid refresh token encoding",
			refreshToken: "invalid_token",
			expectedErr:  customerrors.ErrInvalidJWT,
		},
		{
			name:         "get db refresh token by value error",
			refreshToken: encodedRefreshToken,
			setupMocks: func(
				authService *mockservices.MockAuthService,
				_ *mockservices.MockUsersService,
			) {
				authService.
					EXPECT().
					GetRefreshTokenByValue(gomock.Any(), refreshToken).
					Return(nil, errors.New("db error")).
					Times(1)
			},
			expectedErr: customerrors.ErrInvalidJWT,
		},
		{
			name:         "expire db refresh token error",
			refreshToken: encodedRefreshToken,
			setupMocks: func(
				authService *mockservices.MockAuthService,
				_ *mockservices.MockUsersService,
			) {
				authService.
					EXPECT().
					GetRefreshTokenByValue(gomock.Any(), refreshToken).
					Return(&domains.RefreshToken{ID: 1, UserID: 1, Value: refreshToken}, nil).
					Times(1)

				authService.
					EXPECT().
					ExpireRefreshToken(gomock.Any(), uint64(1)).
					Return(errors.New("test")).
					Times(1)
			},
			expectedErr: errors.New("test"),
		},
		{
			name:         "create db refresh token error",
			refreshToken: encodedRefreshToken,
			setupMocks: func(
				authService *mockservices.MockAuthService,
				_ *mockservices.MockUsersService,
			) {
				authService.
					EXPECT().
					GetRefreshTokenByValue(gomock.Any(), refreshToken).
					Return(&domains.RefreshToken{ID: 1, UserID: 1, Value: refreshToken}, nil).
					Times(1)

				authService.
					EXPECT().
					ExpireRefreshToken(gomock.Any(), uint64(1)).
					Return(nil).
					Times(1)

				authService.
					EXPECT().
					CreateRefreshToken(
						gomock.Any(),
						uint64(1),
						gomock.Any(),
						time.Hour,
					).
					Return(nil, errors.New("test")).
					Times(1)
			},
			expectedErr: errors.New("test"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			authService := mockservices.NewMockAuthService(ctrl)
			usersService := mockservices.NewMockUsersService(ctrl)

			useCases := auth.New(
				authService,
				usersService,
				securityConfig,
				validationConfig,
			)

			if tc.setupMocks != nil {
				tc.setupMocks(
					authService,
					usersService,
				)
			}

			tokens, err := useCases.RefreshTokens(context.Background(), tc.refreshToken)
			if tc.expectedErr != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotZero(t, tokens.AccessToken)
				require.NotZero(t, tokens.RefreshToken)
			}
		})
	}
}

func TestUseCases_LogoutUser(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockAuthService func(*mockservices.MockAuthService)
	}

	type args struct {
		ctx          context.Context
		refreshToken string
	}

	rawRefreshToken := "raw_refresh_token_value"
	encodedRefreshToken := security.RawEncode([]byte(rawRefreshToken))

	dbRefreshToken := &domains.RefreshToken{
		ID:     1,
		UserID: 123,
		Value:  rawRefreshToken,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
	}{
		{
			name: "successfully logout user",
			fields: fields{
				mockAuthService: func(as *mockservices.MockAuthService) {
					as.EXPECT().
						GetRefreshTokenByValue(gomock.Any(), rawRefreshToken).
						Return(dbRefreshToken, nil)
					as.EXPECT().
						ExpireRefreshToken(gomock.Any(), dbRefreshToken.ID).
						Return(nil)
				},
			},
			args: args{
				ctx:          context.Background(),
				refreshToken: encodedRefreshToken,
			},
			wantErr: false,
		},
		{
			name: "invalid encoded refresh token",
			args: args{
				ctx:          context.Background(),
				refreshToken: "!!!invalid-base64!!!",
			},
			wantErr: false, // LogoutUser returns nil on decode error
		},
		{
			name: "refresh token not found in db",
			fields: fields{
				mockAuthService: func(as *mockservices.MockAuthService) {
					as.EXPECT().
						GetRefreshTokenByValue(gomock.Any(), rawRefreshToken).
						Return(nil, errors.New("not found"))
				},
			},
			args: args{
				ctx:          context.Background(),
				refreshToken: encodedRefreshToken,
			},
			wantErr: false, // LogoutUser returns nil when token not found
		},
		{
			name: "error expiring refresh token",
			fields: fields{
				mockAuthService: func(as *mockservices.MockAuthService) {
					as.EXPECT().
						GetRefreshTokenByValue(gomock.Any(), rawRefreshToken).
						Return(dbRefreshToken, nil)
					as.EXPECT().
						ExpireRefreshToken(gomock.Any(), dbRefreshToken.ID).
						Return(errors.New("database error"))
				},
			},
			args: args{
				ctx:          context.Background(),
				refreshToken: encodedRefreshToken,
			},
			wantErr: true,
			err:     errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockAuthService := mockservices.NewMockAuthService(ctrl)
			mockUsersService := mockservices.NewMockUsersService(ctrl)

			if tt.fields.mockAuthService != nil {
				tt.fields.mockAuthService(mockAuthService)
			}

			securityConfig := security.Config{HashCost: hashCost}
			validationConfig := config.ValidationConfig{}

			uc := auth.New(
				mockAuthService,
				mockUsersService,
				securityConfig,
				validationConfig,
			)

			// Act
			err := uc.LogoutUser(tt.args.ctx, tt.args.refreshToken)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					assert.Contains(t, err.Error(), tt.err.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUseCases_LogoutUserFromAllSessions(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockAuthService func(*mockservices.MockAuthService)
	}

	type args struct {
		ctx    context.Context
		userID uint64
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
	}{
		{
			name: "successfully logout from all sessions",
			fields: fields{
				mockAuthService: func(as *mockservices.MockAuthService) {
					as.EXPECT().
						ExpireAllUserRefreshTokens(gomock.Any(), uint64(123)).
						Return(nil)
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 123,
			},
			wantErr: false,
		},
		{
			name: "error expiring all refresh tokens",
			fields: fields{
				mockAuthService: func(as *mockservices.MockAuthService) {
					as.EXPECT().
						ExpireAllUserRefreshTokens(gomock.Any(), uint64(123)).
						Return(errors.New("database error"))
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 123,
			},
			wantErr: true,
			err:     errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockAuthService := mockservices.NewMockAuthService(ctrl)
			mockUsersService := mockservices.NewMockUsersService(ctrl)

			if tt.fields.mockAuthService != nil {
				tt.fields.mockAuthService(mockAuthService)
			}

			securityConfig := security.Config{HashCost: hashCost}
			validationConfig := config.ValidationConfig{}

			uc := auth.New(
				mockAuthService,
				mockUsersService,
				securityConfig,
				validationConfig,
			)

			// Act
			err := uc.LogoutUserFromAllSessions(tt.args.ctx, tt.args.userID)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					assert.Contains(t, err.Error(), tt.err.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUseCases_VerifyEmail(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockAuthService  func(*mockservices.MockAuthService)
		mockUsersService func(*mockservices.MockUsersService)
	}

	type args struct {
		ctx              context.Context
		verifyEmailToken string
	}

	unconfirmedUser := &domains.User{
		ID:             7,
		Email:          "sometest@gmail.com",
		EmailConfirmed: false,
	}

	confirmedUser := &domains.User{
		ID:             7,
		Email:          "sometest@gmail.com",
		EmailConfirmed: true,
	}

	validVerifyToken := security.RawEncode([]byte("salt" + common.SaltSeparator + "7"))

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
	}{
		{
			name: "successfully verify email",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(7)).
						Return(unconfirmedUser, nil)
				},
				mockAuthService: func(as *mockservices.MockAuthService) {
					as.EXPECT().
						VerifyEmail(gomock.Any(), uint64(7)).
						Return(nil)
				},
			},
			args: args{
				ctx:              context.Background(),
				verifyEmailToken: validVerifyToken,
			},
			wantErr: false,
		},
		{
			name: "invalid user ID in token",
			args: args{
				ctx:              context.Background(),
				verifyEmailToken: "invalid",
			},
			wantErr: true,
		},
		{
			name: "token without salt separator",
			args: args{
				ctx:              context.Background(),
				verifyEmailToken: security.RawEncode([]byte("7")),
			},
			wantErr: true,
			err:     customerrors.ErrInvalidJWT,
		},
		{
			name: "user not found",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(7)).
						Return(nil, errors.New("user not found"))
				},
			},
			args: args{
				ctx:              context.Background(),
				verifyEmailToken: validVerifyToken,
			},
			wantErr: true,
			err:     errors.New("user not found"),
		},
		{
			name: "email already confirmed",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(7)).
						Return(confirmedUser, nil)
				},
			},
			args: args{
				ctx:              context.Background(),
				verifyEmailToken: validVerifyToken,
			},
			wantErr: true,
			err:     customerrors.ErrEmailAlreadyConfirmed,
		},
		{
			name: "error verifying email",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(7)).
						Return(unconfirmedUser, nil)
				},
				mockAuthService: func(as *mockservices.MockAuthService) {
					as.EXPECT().
						VerifyEmail(gomock.Any(), uint64(7)).
						Return(errors.New("database error"))
				},
			},
			args: args{
				ctx:              context.Background(),
				verifyEmailToken: validVerifyToken,
			},
			wantErr: true,
			err:     errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockAuthService := mockservices.NewMockAuthService(ctrl)
			mockUsersService := mockservices.NewMockUsersService(ctrl)

			if tt.fields.mockAuthService != nil {
				tt.fields.mockAuthService(mockAuthService)
			}

			if tt.fields.mockUsersService != nil {
				tt.fields.mockUsersService(mockUsersService)
			}

			securityConfig := security.Config{HashCost: hashCost}
			validationConfig := config.ValidationConfig{}

			uc := auth.New(
				mockAuthService,
				mockUsersService,
				securityConfig,
				validationConfig,
			)

			// Act
			err := uc.VerifyEmail(tt.args.ctx, tt.args.verifyEmailToken)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					if errors.Is(tt.err, customerrors.ErrInvalidJWT) ||
						errors.Is(tt.err, customerrors.ErrEmailAlreadyConfirmed) {
						assert.ErrorIs(t, err, tt.err)
					} else {
						assert.Contains(t, err.Error(), tt.err.Error())
					}
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUseCases_ForgetPassword(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockAuthService  func(*mockservices.MockAuthService)
		mockUsersService func(*mockservices.MockUsersService)
	}

	type args struct {
		ctx                 context.Context
		forgetPasswordToken string
		newPassword         string
	}

	pass := "SomeTestP@ssword2"
	hashedPassword, err := security.Hash(pass, hashCost)
	require.NoError(t, err)

	newPass := "SomeTestP@ssword228222"

	user := &domains.User{
		ID:       7,
		Email:    "sometest@gmail.com",
		Password: hashedPassword,
	}

	validForgetToken := security.RawEncode([]byte("salt" + common.SaltSeparator + "7"))

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
	}{
		{
			name: "successfully reset forgotten password",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(7)).
						Return(user, nil)
				},
				mockAuthService: func(as *mockservices.MockAuthService) {
					as.EXPECT().
						ForgetPassword(gomock.Any(), uint64(7), gomock.Any()).
						Return(nil)
				},
			},
			args: args{
				ctx:                 context.Background(),
				forgetPasswordToken: validForgetToken,
				newPassword:         newPass,
			},
			wantErr: false,
		},
		{
			name: "invalid user ID in token",
			args: args{
				ctx:                 context.Background(),
				forgetPasswordToken: "sdadas",
				newPassword:         newPass,
			},
			wantErr: true,
		},
		{
			name: "token without salt separator",
			args: args{
				ctx:                 context.Background(),
				forgetPasswordToken: security.RawEncode([]byte("7")),
				newPassword:         newPass,
			},
			wantErr: true,
			err:     customerrors.ErrInvalidJWT,
		},
		{
			name: "invalid password format",
			args: args{
				ctx:                 context.Background(),
				forgetPasswordToken: validForgetToken,
				newPassword:         "weak",
			},
			wantErr: true,
			err:     customerrors.ErrValidationFailed,
		},
		{
			name: "user not found",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(7)).
						Return(nil, errors.New("user not found"))
				},
			},
			args: args{
				ctx:                 context.Background(),
				forgetPasswordToken: validForgetToken,
				newPassword:         newPass,
			},
			wantErr: true,
			err:     errors.New("user not found"),
		},
		{
			name: "new password same as old password",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(7)).
						Return(user, nil)
				},
			},
			args: args{
				ctx:                 context.Background(),
				forgetPasswordToken: validForgetToken,
				newPassword:         pass,
			},
			wantErr: true,
			err:     customerrors.ErrNewPasswordEqualToOldPassword,
		},
		{
			name: "error updating password in database",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(7)).
						Return(user, nil)
				},
				mockAuthService: func(as *mockservices.MockAuthService) {
					as.EXPECT().
						ForgetPassword(gomock.Any(), uint64(7), gomock.Any()).
						Return(errors.New("database error"))
				},
			},
			args: args{
				ctx:                 context.Background(),
				forgetPasswordToken: validForgetToken,
				newPassword:         newPass,
			},
			wantErr: true,
			err:     errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockAuthService := mockservices.NewMockAuthService(ctrl)
			mockUsersService := mockservices.NewMockUsersService(ctrl)

			if tt.fields.mockAuthService != nil {
				tt.fields.mockAuthService(mockAuthService)
			}

			if tt.fields.mockUsersService != nil {
				tt.fields.mockUsersService(mockUsersService)
			}

			securityConfig := security.Config{HashCost: hashCost}
			validationConfig := config.ValidationConfig{
				PasswordRegExps: []string{
					".{8,}",
					"[a-z]",
					"[A-Z]",
					"[0-9]",
					"[^\\d\\w]",
				},
			}

			uc := auth.New(
				mockAuthService,
				mockUsersService,
				securityConfig,
				validationConfig,
			)

			// Act
			err := uc.ForgetPassword(tt.args.ctx, tt.args.forgetPasswordToken, tt.args.newPassword)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					if errors.Is(tt.err, customerrors.ErrInvalidJWT) ||
						errors.Is(tt.err, customerrors.ErrValidationFailed) {
						assert.ErrorIs(t, err, tt.err)
					} else {
						assert.Contains(t, err.Error(), tt.err.Error())
					}
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUseCases_ChangePassword(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockAuthService  func(*mockservices.MockAuthService)
		mockUsersService func(*mockservices.MockUsersService)
	}

	type args struct {
		ctx context.Context
		dto domains.ChangePasswordDTO
	}

	pass := "SomeTestP@ssword22822"
	hashedPassword, err := security.Hash(pass, hashCost)
	require.NoError(t, err)

	newPass := "SomeTestP@ssword228222"

	user := &domains.User{
		ID:       123,
		Email:    "sometest@gmail.com",
		Password: hashedPassword,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
	}{
		{
			name: "successfully change password",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(123)).
						Return(user, nil)
				},
				mockAuthService: func(as *mockservices.MockAuthService) {
					as.EXPECT().
						ChangePassword(gomock.Any(), uint64(123), gomock.Any()).
						Return(nil)
				},
			},
			args: args{
				ctx: context.Background(),
				dto: domains.ChangePasswordDTO{
					UserID:      123,
					OldPassword: pass,
					NewPassword: newPass,
				},
			},
			wantErr: false,
		},
		{
			name: "new password same as old password",
			args: args{
				ctx: context.Background(),
				dto: domains.ChangePasswordDTO{
					UserID:      123,
					OldPassword: pass,
					NewPassword: pass,
				},
			},
			wantErr: true,
			err:     customerrors.ErrValidationFailed,
		},
		{
			name: "invalid new password format",
			args: args{
				ctx: context.Background(),
				dto: domains.ChangePasswordDTO{
					UserID:      123,
					OldPassword: pass,
					NewPassword: "weak",
				},
			},
			wantErr: true,
			err:     customerrors.ErrValidationFailed,
		},
		{
			name: "user not found",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(999)).
						Return(nil, errors.New("user not found"))
				},
			},
			args: args{
				ctx: context.Background(),
				dto: domains.ChangePasswordDTO{
					UserID:      999,
					OldPassword: pass,
					NewPassword: "NewPass456!",
				},
			},
			wantErr: true,
			err:     errors.New("user not found"),
		},
		{
			name: "wrong old password",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(123)).
						Return(user, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				dto: domains.ChangePasswordDTO{
					UserID:      123,
					OldPassword: "WrongPassword",
					NewPassword: "NewPass456!",
				},
			},
			wantErr: true,
			err:     customerrors.ErrWrongPassword,
		},
		{
			name: "hash function error",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(123)).
						Return(user, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				dto: domains.ChangePasswordDTO{
					UserID:      123,
					OldPassword: "OldPass123!",
					NewPassword: "NewPass456!",
				},
			},
			wantErr: true,
		},
		{
			name: "error updating password in database",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(123)).
						Return(user, nil)
				},
				mockAuthService: func(as *mockservices.MockAuthService) {
					as.EXPECT().
						ChangePassword(gomock.Any(), uint64(123), gomock.Any()).
						Return(errors.New("database error"))
				},
			},
			args: args{
				ctx: context.Background(),
				dto: domains.ChangePasswordDTO{
					UserID:      123,
					OldPassword: pass,
					NewPassword: "SomeTestP@ssword2282222",
				},
			},
			wantErr: true,
			err:     errors.New("database error"),
		},
		{
			name: "empty new password",
			args: args{
				ctx: context.Background(),
				dto: domains.ChangePasswordDTO{
					UserID:      123,
					OldPassword: "OldPass123!",
					NewPassword: "",
				},
			},
			wantErr: true,
			err:     customerrors.ErrValidationFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockAuthService := mockservices.NewMockAuthService(ctrl)
			mockUsersService := mockservices.NewMockUsersService(ctrl)

			if tt.fields.mockAuthService != nil {
				tt.fields.mockAuthService(mockAuthService)
			}

			if tt.fields.mockUsersService != nil {
				tt.fields.mockUsersService(mockUsersService)
			}

			securityConfig := security.Config{HashCost: hashCost}
			validationConfig := config.ValidationConfig{
				PasswordRegExps: []string{
					".{8,}",
					"[a-z]",
					"[A-Z]",
					"[0-9]",
					"[^\\d\\w]",
				},
			}

			uc := auth.New(
				mockAuthService,
				mockUsersService,
				securityConfig,
				validationConfig,
			)

			// Act
			err := uc.ChangePassword(tt.args.ctx, tt.args.dto)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					if errors.Is(tt.err, customerrors.ErrValidationFailed) ||
						errors.Is(tt.err, customerrors.ErrWrongPassword) {
						assert.ErrorIs(t, err, tt.err)
					} else {
						assert.Contains(t, err.Error(), tt.err.Error())
					}
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUseCases_SendVerifyEmailMessage(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockAuthService  func(*mockservices.MockAuthService)
		mockUsersService func(*mockservices.MockUsersService)
	}

	type args struct {
		ctx   context.Context
		email string
	}

	unconfirmedUser := &domains.User{
		ID:             123,
		Email:          "sometest@gmail.com",
		EmailConfirmed: false,
	}

	confirmedUser := &domains.User{
		ID:             123,
		Email:          "sometest@gmail.com",
		EmailConfirmed: true,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
	}{
		{
			name: "successfully send verify email message",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByEmail(gomock.Any(), "sometest@gmail.com").
						Return(unconfirmedUser, nil)
				},
				mockAuthService: func(as *mockservices.MockAuthService) {
					as.EXPECT().
						SendVerifyEmailMessage(gomock.Any(), "sometest@gmail.com").
						Return(nil)
				},
			},
			args: args{
				ctx:   context.Background(),
				email: "sometest@gmail.com",
			},
			wantErr: false,
		},
		{
			name: "user not found by email",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByEmail(gomock.Any(), "nonexistent@gmail.com").
						Return(nil, errors.New("user not found"))
				},
			},
			args: args{
				ctx:   context.Background(),
				email: "nonexistent@gmail.com",
			},
			wantErr: true,
			err:     errors.New("user not found"),
		},
		{
			name: "email already confirmed",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByEmail(gomock.Any(), "sometest@gmail.com").
						Return(confirmedUser, nil)
				},
			},
			args: args{
				ctx:   context.Background(),
				email: "sometest@gmail.com",
			},
			wantErr: true,
			err:     customerrors.ErrEmailAlreadyConfirmed,
		},
		{
			name: "error sending email message",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByEmail(gomock.Any(), "sometest@gmail.com").
						Return(unconfirmedUser, nil)
				},
				mockAuthService: func(as *mockservices.MockAuthService) {
					as.EXPECT().
						SendVerifyEmailMessage(gomock.Any(), "sometest@gmail.com").
						Return(errors.New("SMTP error"))
				},
			},
			args: args{
				ctx:   context.Background(),
				email: "sometest@gmail.com",
			},
			wantErr: true,
			err:     errors.New("SMTP error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockAuthService := mockservices.NewMockAuthService(ctrl)
			mockUsersService := mockservices.NewMockUsersService(ctrl)

			if tt.fields.mockAuthService != nil {
				tt.fields.mockAuthService(mockAuthService)
			}

			if tt.fields.mockUsersService != nil {
				tt.fields.mockUsersService(mockUsersService)
			}

			securityConfig := security.Config{HashCost: hashCost}
			validationConfig := config.ValidationConfig{}

			uc := auth.New(
				mockAuthService,
				mockUsersService,
				securityConfig,
				validationConfig,
			)

			// Act
			err := uc.SendVerifyEmailMessage(tt.args.ctx, tt.args.email)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					if errors.Is(tt.err, customerrors.ErrEmailAlreadyConfirmed) {
						assert.ErrorIs(t, err, tt.err)
					} else {
						assert.Contains(t, err.Error(), tt.err.Error())
					}
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUseCases_SendForgetPasswordMessage(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockAuthService  func(*mockservices.MockAuthService)
		mockUsersService func(*mockservices.MockUsersService)
	}

	type args struct {
		ctx   context.Context
		email string
	}

	confirmedUser := &domains.User{
		ID:             123,
		Email:          "sometest@gmail.com",
		EmailConfirmed: true,
	}

	unconfirmedUser := &domains.User{
		ID:             123,
		Email:          "sometest@gmail.com",
		EmailConfirmed: false,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
	}{
		{
			name: "successfully send forget password message",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByEmail(gomock.Any(), "sometest@gmail.com").
						Return(confirmedUser, nil)
				},
				mockAuthService: func(as *mockservices.MockAuthService) {
					as.EXPECT().
						SendForgetPasswordMessage(gomock.Any(), "sometest@gmail.com").
						Return(nil)
				},
			},
			args: args{
				ctx:   context.Background(),
				email: "sometest@gmail.com",
			},
			wantErr: false,
		},
		{
			name: "user not found by email",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByEmail(gomock.Any(), "nonexistent@gmail.com").
						Return(nil, errors.New("user not found"))
				},
			},
			args: args{
				ctx:   context.Background(),
				email: "nonexistent@gmail.com",
			},
			wantErr: true,
			err:     errors.New("user not found"),
		},
		{
			name: "email not confirmed",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByEmail(gomock.Any(), "sometest@gmail.com").
						Return(unconfirmedUser, nil)
				},
			},
			args: args{
				ctx:   context.Background(),
				email: "sometest@gmail.com",
			},
			wantErr: true,
			err:     customerrors.ErrEmailNotConfirmed,
		},
		{
			name: "error sending email message",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByEmail(gomock.Any(), "sometest@gmail.com").
						Return(confirmedUser, nil)
				},
				mockAuthService: func(as *mockservices.MockAuthService) {
					as.EXPECT().
						SendForgetPasswordMessage(gomock.Any(), "sometest@gmail.com").
						Return(errors.New("SMTP error"))
				},
			},
			args: args{
				ctx:   context.Background(),
				email: "sometest@gmail.com",
			},
			wantErr: true,
			err:     errors.New("SMTP error"),
		},
		{
			name: "empty email",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByEmail(gomock.Any(), "").
						Return(nil, customerrors.ErrUserNotFound)
				},
			},
			args: args{
				ctx:   context.Background(),
				email: "",
			},
			wantErr: true,
			err:     errors.New("user not found"), // GetUserByEmail вернет ошибку
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockAuthService := mockservices.NewMockAuthService(ctrl)
			mockUsersService := mockservices.NewMockUsersService(ctrl)

			if tt.fields.mockAuthService != nil {
				tt.fields.mockAuthService(mockAuthService)
			}

			if tt.fields.mockUsersService != nil {
				tt.fields.mockUsersService(mockUsersService)
			}

			securityConfig := security.Config{HashCost: hashCost}
			validationConfig := config.ValidationConfig{}

			uc := auth.New(
				mockAuthService,
				mockUsersService,
				securityConfig,
				validationConfig,
			)

			// Act
			err := uc.SendForgetPasswordMessage(tt.args.ctx, tt.args.email)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					if errors.Is(tt.err, customerrors.ErrEmailNotConfirmed) {
						assert.ErrorIs(t, err, tt.err)
					} else {
						assert.Contains(t, err.Error(), tt.err.Error())
					}
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
