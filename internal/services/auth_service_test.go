package services_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/services"
	mockrepositories "github.com/DKhorkov/kfc/mocks/repositories"
	mockuow "github.com/DKhorkov/kfc/mocks/uow"
	pg "github.com/DKhorkov/libs/db/postgresql"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestAuthService_RegisterUser(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW              func(*mockuow.MockUnitOfWork)
		mockAuthRepository   func(*mockrepositories.MockAuthRepository)
		mockUsersRepository  func(*mockrepositories.MockUsersRepository)
		mockEmailsRepository func(*mockrepositories.MockEmailsRepository)
	}

	type args struct {
		ctx      context.Context
		userData domains.RegisterDTO
	}

	newUser := &domains.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
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
			name: "successfully register new user",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							// Создаем mock транзакцию
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockUsersRepository: func(ur *mockrepositories.MockUsersRepository) {
					// Первая проверка - email не существует
					ur.EXPECT().
						GetUserByEmail(gomock.Any(), "test@example.com").
						Return(nil, sql.ErrNoRows)

					// Вторая проверка - username не существует
					ur.EXPECT().
						GetUserByUsername(gomock.Any(), "test@example.com").
						Return(nil, sql.ErrNoRows)

					// После регистрации получаем пользователя
					ur.EXPECT().
						GetUserByEmail(gomock.Any(), "test@example.com").
						Return(newUser, nil)
				},
				mockAuthRepository: func(ar *mockrepositories.MockAuthRepository) {
					ar.EXPECT().
						RegisterUser(gomock.Any(), gomock.AssignableToTypeOf(domains.RegisterDTO{})).
						Return(uint64(1), nil)
				},
				mockEmailsRepository: func(er *mockrepositories.MockEmailsRepository) {
					er.EXPECT().
						SendVerifyEmailMessage(gomock.Any(), *newUser).
						Return(nil)
				},
			},
			args: args{
				ctx: context.Background(),
				userData: domains.RegisterDTO{
					Username: "testuser",
					Email:    "test@example.com",
					Password: "Password123!",
				},
			},
			want:    newUser,
			wantErr: false,
		},
		{
			name: "user with email already exists",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockUsersRepository: func(ur *mockrepositories.MockUsersRepository) {
					ur.EXPECT().
						GetUserByEmail(gomock.Any(), "existing@example.com").
						Return(&domains.User{ID: 1}, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				userData: domains.RegisterDTO{
					Username: "testuser",
					Email:    "existing@example.com",
					Password: "Password123!",
				},
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrUserAlreadyExists,
		},
		{
			name: "user with username already exists",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockUsersRepository: func(ur *mockrepositories.MockUsersRepository) {
					ur.EXPECT().
						GetUserByEmail(gomock.Any(), "test@example.com").
						Return(nil, sql.ErrNoRows)

					ur.EXPECT().
						GetUserByUsername(gomock.Any(), "test@example.com").
						Return(&domains.User{ID: 1}, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				userData: domains.RegisterDTO{
					Username: "testuser",
					Email:    "test@example.com",
					Password: "Password123!",
				},
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrUserAlreadyExists,
		},
		{
			name: "error registering user in repository",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockUsersRepository: func(ur *mockrepositories.MockUsersRepository) {
					ur.EXPECT().
						GetUserByEmail(gomock.Any(), "test@example.com").
						Return(nil, sql.ErrNoRows)

					ur.EXPECT().
						GetUserByUsername(gomock.Any(), "test@example.com").
						Return(nil, sql.ErrNoRows)
				},
				mockAuthRepository: func(ar *mockrepositories.MockAuthRepository) {
					ar.EXPECT().
						RegisterUser(gomock.Any(), gomock.Any()).
						Return(uint64(0), errors.New("database error"))
				},
			},
			args: args{
				ctx: context.Background(),
				userData: domains.RegisterDTO{
					Username: "testuser",
					Email:    "test@example.com",
					Password: "Password123!",
				},
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("database error"),
		},
		{
			name: "error getting user after registration",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockUsersRepository: func(ur *mockrepositories.MockUsersRepository) {
					ur.EXPECT().
						GetUserByEmail(gomock.Any(), "test@example.com").
						Return(nil, sql.ErrNoRows)

					ur.EXPECT().
						GetUserByUsername(gomock.Any(), "test@example.com").
						Return(nil, sql.ErrNoRows)

					ur.EXPECT().
						GetUserByEmail(gomock.Any(), "test@example.com").
						Return(nil, errors.New("user not found"))
				},
				mockAuthRepository: func(ar *mockrepositories.MockAuthRepository) {
					ar.EXPECT().
						RegisterUser(gomock.Any(), gomock.Any()).
						Return(uint64(1), nil)
				},
			},
			args: args{
				ctx: context.Background(),
				userData: domains.RegisterDTO{
					Username: "testuser",
					Email:    "test@example.com",
					Password: "Password123!",
				},
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("user not found"),
		},
		{
			name: "error sending verification email",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockUsersRepository: func(ur *mockrepositories.MockUsersRepository) {
					ur.EXPECT().
						GetUserByEmail(gomock.Any(), "test@example.com").
						Return(nil, sql.ErrNoRows)

					ur.EXPECT().
						GetUserByUsername(gomock.Any(), "test@example.com").
						Return(nil, sql.ErrNoRows)

					ur.EXPECT().
						GetUserByEmail(gomock.Any(), "test@example.com").
						Return(newUser, nil)
				},
				mockAuthRepository: func(ar *mockrepositories.MockAuthRepository) {
					ar.EXPECT().
						RegisterUser(gomock.Any(), gomock.Any()).
						Return(uint64(1), nil)
				},
				mockEmailsRepository: func(er *mockrepositories.MockEmailsRepository) {
					er.EXPECT().
						SendVerifyEmailMessage(gomock.Any(), *newUser).
						Return(errors.New("SMTP error"))
				},
			},
			args: args{
				ctx: context.Background(),
				userData: domains.RegisterDTO{
					Username: "testuser",
					Email:    "test@example.com",
					Password: "Password123!",
				},
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("SMTP error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockUOW := mockuow.NewMockUnitOfWork(ctrl)
			mockAuthRepo := mockrepositories.NewMockAuthRepository(ctrl)
			mockUsersRepo := mockrepositories.NewMockUsersRepository(ctrl)
			mockEmailsRepo := mockrepositories.NewMockEmailsRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockAuthRepository != nil {
				tt.fields.mockAuthRepository(mockAuthRepo)
			}

			if tt.fields.mockUsersRepository != nil {
				tt.fields.mockUsersRepository(mockUsersRepo)
			}

			if tt.fields.mockEmailsRepository != nil {
				tt.fields.mockEmailsRepository(mockEmailsRepo)
			}

			// Создаем factory функции
			newAuthRepoFunc := func(_ pg.Transaction) interfaces.AuthRepository {
				return mockAuthRepo
			}

			newUsersRepoFunc := func(_ pg.Transaction) interfaces.UsersRepository {
				return mockUsersRepo
			}

			newEmailsRepoFunc := func() interfaces.EmailsRepository {
				return mockEmailsRepo
			}

			service := services.NewAuthService(
				mockUOW,
				newAuthRepoFunc,
				newUsersRepoFunc,
				newEmailsRepoFunc,
			)

			// Act
			got, err := service.RegisterUser(tt.args.ctx, tt.args.userData)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					if errors.Is(tt.err, customerrors.ErrUserAlreadyExists) {
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

func TestAuthService_CreateRefreshToken(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW            func(*mockuow.MockUnitOfWork)
		mockAuthRepository func(*mockrepositories.MockAuthRepository)
	}

	type args struct {
		ctx    context.Context
		userID uint64
		value  string
		ttl    time.Duration
	}

	refreshToken := &domains.RefreshToken{
		ID:     1,
		UserID: 123,
		Value:  "refresh_token_value",
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domains.RefreshToken
		wantErr bool
		err     error
	}{
		{
			name: "successfully create refresh token",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockAuthRepository: func(ar *mockrepositories.MockAuthRepository) {
					ar.EXPECT().
						CreateRefreshToken(gomock.Any(), uint64(123), "token_value", time.Hour).
						Return(uint64(1), nil)

					ar.EXPECT().
						GetRefreshTokenByUserID(gomock.Any(), uint64(123)).
						Return(refreshToken, nil)
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 123,
				value:  "token_value",
				ttl:    time.Hour,
			},
			want:    refreshToken,
			wantErr: false,
		},
		{
			name: "error creating refresh token",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockAuthRepository: func(ar *mockrepositories.MockAuthRepository) {
					ar.EXPECT().
						CreateRefreshToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(uint64(0), errors.New("database error"))
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 123,
				value:  "token_value",
				ttl:    time.Hour,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("database error"),
		},
		{
			name: "error getting refresh token after creation",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockAuthRepository: func(ar *mockrepositories.MockAuthRepository) {
					ar.EXPECT().
						CreateRefreshToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(uint64(1), nil)

					ar.EXPECT().
						GetRefreshTokenByUserID(gomock.Any(), gomock.Any()).
						Return(nil, errors.New("not found"))
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 123,
				value:  "token_value",
				ttl:    time.Hour,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockUOW := mockuow.NewMockUnitOfWork(ctrl)
			mockAuthRepo := mockrepositories.NewMockAuthRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockAuthRepository != nil {
				tt.fields.mockAuthRepository(mockAuthRepo)
			}

			newAuthRepoFunc := func(_ pg.Transaction) interfaces.AuthRepository {
				return mockAuthRepo
			}

			service := services.NewAuthService(
				mockUOW,
				newAuthRepoFunc,
				nil,
				nil,
			)

			// Act
			got, err := service.CreateRefreshToken(
				tt.args.ctx,
				tt.args.userID,
				tt.args.value,
				tt.args.ttl,
			)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					assert.Contains(t, err.Error(), tt.err.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestAuthService_GetRefreshTokenByUserID(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW            func(*mockuow.MockUnitOfWork)
		mockAuthRepository func(*mockrepositories.MockAuthRepository)
	}

	type args struct {
		ctx    context.Context
		userID uint64
	}

	refreshToken := &domains.RefreshToken{
		ID:     1,
		UserID: 123,
		Value:  "refresh_token_value",
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domains.RefreshToken
		wantErr bool
		err     error
	}{
		{
			name: "successfully get refresh token",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockAuthRepository: func(ar *mockrepositories.MockAuthRepository) {
					ar.EXPECT().
						GetRefreshTokenByUserID(gomock.Any(), uint64(123)).
						Return(refreshToken, nil)
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 123,
			},
			want:    refreshToken,
			wantErr: false,
		},
		{
			name: "refresh token not found",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockAuthRepository: func(ar *mockrepositories.MockAuthRepository) {
					ar.EXPECT().
						GetRefreshTokenByUserID(gomock.Any(), uint64(999)).
						Return(nil, sql.ErrNoRows)
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 999,
			},
			want:    nil,
			wantErr: true,
			err:     sql.ErrNoRows,
		},
		{
			name: "database error",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockAuthRepository: func(ar *mockrepositories.MockAuthRepository) {
					ar.EXPECT().
						GetRefreshTokenByUserID(gomock.Any(), uint64(123)).
						Return(nil, errors.New("database connection failed"))
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 123,
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockUOW := mockuow.NewMockUnitOfWork(ctrl)
			mockAuthRepo := mockrepositories.NewMockAuthRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockAuthRepository != nil {
				tt.fields.mockAuthRepository(mockAuthRepo)
			}

			newAuthRepoFunc := func(_ pg.Transaction) interfaces.AuthRepository {
				return mockAuthRepo
			}

			service := services.NewAuthService(
				mockUOW,
				newAuthRepoFunc,
				nil,
				nil,
			)

			// Act
			got, err := service.GetRefreshTokenByUserID(tt.args.ctx, tt.args.userID)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					assert.ErrorIs(t, err, tt.err)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestAuthService_ExpireRefreshToken(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW            func(*mockuow.MockUnitOfWork)
		mockAuthRepository func(*mockrepositories.MockAuthRepository)
	}

	type args struct {
		ctx            context.Context
		refreshTokenID uint64
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
	}{
		{
			name: "successfully expire refresh token",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockAuthRepository: func(ar *mockrepositories.MockAuthRepository) {
					ar.EXPECT().
						ExpireRefreshToken(gomock.Any(), uint64(1)).
						Return(nil)
				},
			},
			args: args{
				ctx:            context.Background(),
				refreshTokenID: 1,
			},
			wantErr: false,
		},
		{
			name: "error expiring refresh token",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockAuthRepository: func(ar *mockrepositories.MockAuthRepository) {
					ar.EXPECT().
						ExpireRefreshToken(gomock.Any(), uint64(1)).
						Return(errors.New("database error"))
				},
			},
			args: args{
				ctx:            context.Background(),
				refreshTokenID: 1,
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

			mockUOW := mockuow.NewMockUnitOfWork(ctrl)
			mockAuthRepo := mockrepositories.NewMockAuthRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockAuthRepository != nil {
				tt.fields.mockAuthRepository(mockAuthRepo)
			}

			newAuthRepoFunc := func(_ pg.Transaction) interfaces.AuthRepository {
				return mockAuthRepo
			}

			service := services.NewAuthService(
				mockUOW,
				newAuthRepoFunc,
				nil,
				nil,
			)

			// Act
			err := service.ExpireRefreshToken(tt.args.ctx, tt.args.refreshTokenID)

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

func TestAuthService_VerifyEmail(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW            func(*mockuow.MockUnitOfWork)
		mockAuthRepository func(*mockrepositories.MockAuthRepository)
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
			name: "successfully verify email",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockAuthRepository: func(ar *mockrepositories.MockAuthRepository) {
					ar.EXPECT().
						VerifyEmail(gomock.Any(), uint64(123)).
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
			name: "error verifying email",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockAuthRepository: func(ar *mockrepositories.MockAuthRepository) {
					ar.EXPECT().
						VerifyEmail(gomock.Any(), uint64(123)).
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

			mockUOW := mockuow.NewMockUnitOfWork(ctrl)
			mockAuthRepo := mockrepositories.NewMockAuthRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockAuthRepository != nil {
				tt.fields.mockAuthRepository(mockAuthRepo)
			}

			newAuthRepoFunc := func(_ pg.Transaction) interfaces.AuthRepository {
				return mockAuthRepo
			}

			service := services.NewAuthService(
				mockUOW,
				newAuthRepoFunc,
				nil,
				nil,
			)

			// Act
			err := service.VerifyEmail(tt.args.ctx, tt.args.userID)

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

func TestAuthService_ForgetPassword(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW            func(*mockuow.MockUnitOfWork)
		mockAuthRepository func(*mockrepositories.MockAuthRepository)
	}

	type args struct {
		ctx         context.Context
		userID      uint64
		newPassword string
	}

	refreshToken := &domains.RefreshToken{
		ID:     1,
		UserID: 123,
		Value:  "refresh_token_value",
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
	}{
		{
			name: "successfully reset password with existing refresh token",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockAuthRepository: func(ar *mockrepositories.MockAuthRepository) {
					ar.EXPECT().
						ChangePassword(gomock.Any(), uint64(123), "new_hashed_password").
						Return(nil)

					ar.EXPECT().
						GetRefreshTokenByUserID(gomock.Any(), uint64(123)).
						Return(refreshToken, nil)

					ar.EXPECT().
						ExpireRefreshToken(gomock.Any(), refreshToken.ID).
						Return(nil)
				},
			},
			args: args{
				ctx:         context.Background(),
				userID:      123,
				newPassword: "new_hashed_password",
			},
			wantErr: false,
		},
		{
			name: "successfully reset password without refresh token",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockAuthRepository: func(ar *mockrepositories.MockAuthRepository) {
					ar.EXPECT().
						ChangePassword(gomock.Any(), uint64(123), "new_hashed_password").
						Return(nil)

					ar.EXPECT().
						GetRefreshTokenByUserID(gomock.Any(), uint64(123)).
						Return(nil, sql.ErrNoRows)
					// No ExpireRefreshToken call expected
				},
			},
			args: args{
				ctx:         context.Background(),
				userID:      123,
				newPassword: "new_hashed_password",
			},
			wantErr: false,
		},
		{
			name: "error changing password",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockAuthRepository: func(ar *mockrepositories.MockAuthRepository) {
					ar.EXPECT().
						ChangePassword(gomock.Any(), uint64(123), "new_hashed_password").
						Return(errors.New("database error"))
				},
			},
			args: args{
				ctx:         context.Background(),
				userID:      123,
				newPassword: "new_hashed_password",
			},
			wantErr: true,
			err:     errors.New("database error"),
		},
		{
			name: "error getting refresh token",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockAuthRepository: func(ar *mockrepositories.MockAuthRepository) {
					ar.EXPECT().
						ChangePassword(gomock.Any(), uint64(123), "new_hashed_password").
						Return(nil)

					ar.EXPECT().
						GetRefreshTokenByUserID(gomock.Any(), uint64(123)).
						Return(nil, errors.New("database connection failed"))
				},
			},
			args: args{
				ctx:         context.Background(),
				userID:      123,
				newPassword: "new_hashed_password",
			},
			wantErr: true,
			err:     errors.New("database connection failed"),
		},
		{
			name: "error expiring refresh token",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockAuthRepository: func(ar *mockrepositories.MockAuthRepository) {
					ar.EXPECT().
						ChangePassword(gomock.Any(), uint64(123), "new_hashed_password").
						Return(nil)

					ar.EXPECT().
						GetRefreshTokenByUserID(gomock.Any(), uint64(123)).
						Return(refreshToken, nil)

					ar.EXPECT().
						ExpireRefreshToken(gomock.Any(), refreshToken.ID).
						Return(errors.New("expire error"))
				},
			},
			args: args{
				ctx:         context.Background(),
				userID:      123,
				newPassword: "new_hashed_password",
			},
			wantErr: true,
			err:     errors.New("expire error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockUOW := mockuow.NewMockUnitOfWork(ctrl)
			mockAuthRepo := mockrepositories.NewMockAuthRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockAuthRepository != nil {
				tt.fields.mockAuthRepository(mockAuthRepo)
			}

			newAuthRepoFunc := func(_ pg.Transaction) interfaces.AuthRepository {
				return mockAuthRepo
			}

			service := services.NewAuthService(
				mockUOW,
				newAuthRepoFunc,
				nil,
				nil,
			)

			// Act
			err := service.ForgetPassword(tt.args.ctx, tt.args.userID, tt.args.newPassword)

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

func TestAuthService_ChangePassword(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW            func(*mockuow.MockUnitOfWork)
		mockAuthRepository func(*mockrepositories.MockAuthRepository)
	}

	type args struct {
		ctx         context.Context
		userID      uint64
		newPassword string
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
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockAuthRepository: func(ar *mockrepositories.MockAuthRepository) {
					ar.EXPECT().
						ChangePassword(gomock.Any(), uint64(123), "new_hashed_password").
						Return(nil)
				},
			},
			args: args{
				ctx:         context.Background(),
				userID:      123,
				newPassword: "new_hashed_password",
			},
			wantErr: false,
		},
		{
			name: "error changing password",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockAuthRepository: func(ar *mockrepositories.MockAuthRepository) {
					ar.EXPECT().
						ChangePassword(gomock.Any(), uint64(123), "new_hashed_password").
						Return(errors.New("database error"))
				},
			},
			args: args{
				ctx:         context.Background(),
				userID:      123,
				newPassword: "new_hashed_password",
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

			mockUOW := mockuow.NewMockUnitOfWork(ctrl)
			mockAuthRepo := mockrepositories.NewMockAuthRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockAuthRepository != nil {
				tt.fields.mockAuthRepository(mockAuthRepo)
			}

			newAuthRepoFunc := func(_ pg.Transaction) interfaces.AuthRepository {
				return mockAuthRepo
			}

			service := services.NewAuthService(
				mockUOW,
				newAuthRepoFunc,
				nil,
				nil,
			)

			// Act
			err := service.ChangePassword(tt.args.ctx, tt.args.userID, tt.args.newPassword)

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

func TestAuthService_SendForgetPasswordMessage(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW              func(*mockuow.MockUnitOfWork)
		mockUsersRepository  func(*mockrepositories.MockUsersRepository)
		mockEmailsRepository func(*mockrepositories.MockEmailsRepository)
	}

	type args struct {
		ctx   context.Context
		email string
	}

	user := &domains.User{
		ID:    123,
		Email: "test@example.com",
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
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockUsersRepository: func(ur *mockrepositories.MockUsersRepository) {
					ur.EXPECT().
						GetUserByEmail(gomock.Any(), "test@example.com").
						Return(user, nil)
				},
				mockEmailsRepository: func(er *mockrepositories.MockEmailsRepository) {
					er.EXPECT().
						SendForgetPasswordMessage(gomock.Any(), *user).
						Return(nil)
				},
			},
			args: args{
				ctx:   context.Background(),
				email: "test@example.com",
			},
			wantErr: false,
		},
		{
			name: "user not found",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockUsersRepository: func(ur *mockrepositories.MockUsersRepository) {
					ur.EXPECT().
						GetUserByEmail(gomock.Any(), "nonexistent@example.com").
						Return(nil, sql.ErrNoRows)
				},
			},
			args: args{
				ctx:   context.Background(),
				email: "nonexistent@example.com",
			},
			wantErr: true,
			err:     customerrors.ErrUserNotFound,
		},
		{
			name: "error sending email",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockUsersRepository: func(ur *mockrepositories.MockUsersRepository) {
					ur.EXPECT().
						GetUserByEmail(gomock.Any(), "test@example.com").
						Return(user, nil)
				},
				mockEmailsRepository: func(er *mockrepositories.MockEmailsRepository) {
					er.EXPECT().
						SendForgetPasswordMessage(gomock.Any(), *user).
						Return(errors.New("SMTP error"))
				},
			},
			args: args{
				ctx:   context.Background(),
				email: "test@example.com",
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

			mockUOW := mockuow.NewMockUnitOfWork(ctrl)
			mockUsersRepo := mockrepositories.NewMockUsersRepository(ctrl)
			mockEmailsRepo := mockrepositories.NewMockEmailsRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockUsersRepository != nil {
				tt.fields.mockUsersRepository(mockUsersRepo)
			}

			if tt.fields.mockEmailsRepository != nil {
				tt.fields.mockEmailsRepository(mockEmailsRepo)
			}

			newUsersRepoFunc := func(_ pg.Transaction) interfaces.UsersRepository {
				return mockUsersRepo
			}

			newEmailsRepoFunc := func() interfaces.EmailsRepository {
				return mockEmailsRepo
			}

			service := services.NewAuthService(
				mockUOW,
				nil,
				newUsersRepoFunc,
				newEmailsRepoFunc,
			)

			// Act
			err := service.SendForgetPasswordMessage(tt.args.ctx, tt.args.email)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					if errors.Is(tt.err, customerrors.ErrUserNotFound) {
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

func TestAuthService_SendVerifyEmailMessage(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW              func(*mockuow.MockUnitOfWork)
		mockUsersRepository  func(*mockrepositories.MockUsersRepository)
		mockEmailsRepository func(*mockrepositories.MockEmailsRepository)
	}

	type args struct {
		ctx   context.Context
		email string
	}

	user := &domains.User{
		ID:    123,
		Email: "test@example.com",
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
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockUsersRepository: func(ur *mockrepositories.MockUsersRepository) {
					ur.EXPECT().
						GetUserByEmail(gomock.Any(), "test@example.com").
						Return(user, nil)
				},
				mockEmailsRepository: func(er *mockrepositories.MockEmailsRepository) {
					er.EXPECT().
						SendVerifyEmailMessage(gomock.Any(), *user).
						Return(nil)
				},
			},
			args: args{
				ctx:   context.Background(),
				email: "test@example.com",
			},
			wantErr: false,
		},
		{
			name: "user not found",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockUsersRepository: func(ur *mockrepositories.MockUsersRepository) {
					ur.EXPECT().
						GetUserByEmail(gomock.Any(), "nonexistent@example.com").
						Return(nil, sql.ErrNoRows)
				},
			},
			args: args{
				ctx:   context.Background(),
				email: "nonexistent@example.com",
			},
			wantErr: true,
			err:     customerrors.ErrUserNotFound,
		},
		{
			name: "error sending email",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockUsersRepository: func(ur *mockrepositories.MockUsersRepository) {
					ur.EXPECT().
						GetUserByEmail(gomock.Any(), "test@example.com").
						Return(user, nil)
				},
				mockEmailsRepository: func(er *mockrepositories.MockEmailsRepository) {
					er.EXPECT().
						SendVerifyEmailMessage(gomock.Any(), *user).
						Return(errors.New("SMTP error"))
				},
			},
			args: args{
				ctx:   context.Background(),
				email: "test@example.com",
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

			mockUOW := mockuow.NewMockUnitOfWork(ctrl)
			mockUsersRepo := mockrepositories.NewMockUsersRepository(ctrl)
			mockEmailsRepo := mockrepositories.NewMockEmailsRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockUsersRepository != nil {
				tt.fields.mockUsersRepository(mockUsersRepo)
			}

			if tt.fields.mockEmailsRepository != nil {
				tt.fields.mockEmailsRepository(mockEmailsRepo)
			}

			newUsersRepoFunc := func(_ pg.Transaction) interfaces.UsersRepository {
				return mockUsersRepo
			}

			newEmailsRepoFunc := func() interfaces.EmailsRepository {
				return mockEmailsRepo
			}

			service := services.NewAuthService(
				mockUOW,
				nil,
				newUsersRepoFunc,
				newEmailsRepoFunc,
			)

			// Act
			err := service.SendVerifyEmailMessage(tt.args.ctx, tt.args.email)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					if errors.Is(tt.err, customerrors.ErrUserNotFound) {
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
