package users_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	service "github.com/DKhorkov/kfc/internal/services/users"
	mockrepositories "github.com/DKhorkov/kfc/mocks/repositories"
	mockuow "github.com/DKhorkov/kfc/mocks/uow"
	pg "github.com/DKhorkov/libs/db/postgresql"
	"github.com/DKhorkov/libs/pointers"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestService_GetUsers(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW             func(*mockuow.MockUnitOfWork)
		mockUsersRepository func(*mockrepositories.MockUsersRepository)
	}

	type args struct {
		ctx        context.Context
		filters    *domains.UsersFilters
		pagination *domains.Pagination
	}

	users := []domains.User{
		{ID: 1, Username: "user1", Email: "user1@example.com"},
		{ID: 2, Username: "user2", Email: "user2@example.com"},
	}

	pagination := &domains.Pagination{
		Limit:  pointers.New[uint64](10),
		Offset: pointers.New[uint64](0),
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []domains.User
		wantErr bool
		err     error
	}{
		{
			name: "successfully get users with pagination",
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
						GetUsers(gomock.Any(), (*domains.UsersFilters)(nil), pagination).
						Return(users, nil)
				},
			},
			args: args{
				ctx:        context.Background(),
				filters:    nil,
				pagination: pagination,
			},
			want:    users,
			wantErr: false,
		},
		{
			name: "successfully get users with username filter",
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
					filters := &domains.UsersFilters{Username: pointers.New("test")}
					ur.EXPECT().
						GetUsers(gomock.Any(), filters, pagination).
						Return([]domains.User{{ID: 1, Username: "testuser"}}, nil)
				},
			},
			args: args{
				ctx:        context.Background(),
				filters:    &domains.UsersFilters{Username: pointers.New("test")},
				pagination: pagination,
			},
			want:    []domains.User{{ID: 1, Username: "testuser"}},
			wantErr: false,
		},
		{
			name: "successfully get users without pagination",
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
						GetUsers(gomock.Any(), (*domains.UsersFilters)(nil), (*domains.Pagination)(nil)).
						Return(users, nil)
				},
			},
			args: args{
				ctx:        context.Background(),
				filters:    nil,
				pagination: nil,
			},
			want:    users,
			wantErr: false,
		},
		{
			name: "empty users list",
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
						GetUsers(gomock.Any(), gomock.Any(), gomock.Any()).
						Return([]domains.User{}, nil)
				},
			},
			args: args{
				ctx:        context.Background(),
				filters:    &domains.UsersFilters{Username: pointers.New("nonexistent")},
				pagination: pagination,
			},
			want:    []domains.User{},
			wantErr: false,
		},
		{
			name: "error getting users",
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
						GetUsers(gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil, errors.New("database error"))
				},
			},
			args: args{
				ctx:        context.Background(),
				filters:    nil,
				pagination: pagination,
			},
			want:    nil,
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
			mockUsersRepo := mockrepositories.NewMockUsersRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockUsersRepository != nil {
				tt.fields.mockUsersRepository(mockUsersRepo)
			}

			newUsersRepoFunc := func(_ pg.Transaction) interfaces.UsersRepository {
				return mockUsersRepo
			}

			s := service.New(mockUOW, newUsersRepoFunc)

			// Act
			got, err := s.GetUsers(tt.args.ctx, tt.args.filters, tt.args.pagination)

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

func TestService_GetUserByID(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW             func(*mockuow.MockUnitOfWork)
		mockUsersRepository func(*mockrepositories.MockUsersRepository)
	}

	type args struct {
		ctx context.Context
		id  uint64
	}

	user := &domains.User{
		ID:             1,
		Username:       "testuser",
		Email:          "test@example.com",
		EmailConfirmed: true,
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
			name: "successfully get user by ID",
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
						GetUserByID(gomock.Any(), uint64(1)).
						Return(user, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				id:  1,
			},
			want:    user,
			wantErr: false,
		},
		{
			name: "user not found by ID",
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
						GetUserByID(gomock.Any(), uint64(999)).
						Return(nil, sql.ErrNoRows)
				},
			},
			args: args{
				ctx: context.Background(),
				id:  999,
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrUserNotFound,
		},
		{
			name: "database error getting user by ID",
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
						GetUserByID(gomock.Any(), uint64(1)).
						Return(nil, errors.New("database connection failed"))
				},
			},
			args: args{
				ctx: context.Background(),
				id:  1,
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrUserNotFound,
		},
		{
			name: "get user with zero ID",
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
						GetUserByID(gomock.Any(), uint64(0)).
						Return(nil, sql.ErrNoRows)
				},
			},
			args: args{
				ctx: context.Background(),
				id:  0,
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockUOW := mockuow.NewMockUnitOfWork(ctrl)
			mockUsersRepo := mockrepositories.NewMockUsersRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockUsersRepository != nil {
				tt.fields.mockUsersRepository(mockUsersRepo)
			}

			newUsersRepoFunc := func(_ pg.Transaction) interfaces.UsersRepository {
				return mockUsersRepo
			}

			s := service.New(mockUOW, newUsersRepoFunc)

			// Act
			got, err := s.GetUserByID(tt.args.ctx, tt.args.id)

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

func TestService_GetUserByEmail(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW             func(*mockuow.MockUnitOfWork)
		mockUsersRepository func(*mockrepositories.MockUsersRepository)
	}

	type args struct {
		ctx   context.Context
		email string
	}

	user := &domains.User{
		ID:             1,
		Username:       "testuser",
		Email:          "test@example.com",
		EmailConfirmed: true,
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
			name: "successfully get user by email",
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
			},
			args: args{
				ctx:   context.Background(),
				email: "test@example.com",
			},
			want:    user,
			wantErr: false,
		},
		{
			name: "user not found by email",
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
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrUserNotFound,
		},
		{
			name: "database error getting user by email",
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
						Return(nil, errors.New("database error"))
				},
			},
			args: args{
				ctx:   context.Background(),
				email: "test@example.com",
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrUserNotFound,
		},
		{
			name: "empty email",
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
						GetUserByEmail(gomock.Any(), "").
						Return(nil, sql.ErrNoRows)
				},
			},
			args: args{
				ctx:   context.Background(),
				email: "",
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrUserNotFound,
		},
		{
			name: "email with special characters",
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
						GetUserByEmail(gomock.Any(), "user+tag@example.com").
						Return(user, nil)
				},
			},
			args: args{
				ctx:   context.Background(),
				email: "user+tag@example.com",
			},
			want:    user,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockUOW := mockuow.NewMockUnitOfWork(ctrl)
			mockUsersRepo := mockrepositories.NewMockUsersRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockUsersRepository != nil {
				tt.fields.mockUsersRepository(mockUsersRepo)
			}

			newUsersRepoFunc := func(_ pg.Transaction) interfaces.UsersRepository {
				return mockUsersRepo
			}

			s := service.New(mockUOW, newUsersRepoFunc)

			// Act
			got, err := s.GetUserByEmail(tt.args.ctx, tt.args.email)

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

func TestService_GetUserByUsername(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW             func(*mockuow.MockUnitOfWork)
		mockUsersRepository func(*mockrepositories.MockUsersRepository)
	}

	type args struct {
		ctx      context.Context
		username string
	}

	user := &domains.User{
		ID:             1,
		Username:       "testuser",
		Email:          "test@example.com",
		EmailConfirmed: true,
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
			name: "successfully get user by username",
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
						GetUserByUsername(gomock.Any(), "testuser").
						Return(user, nil)
				},
			},
			args: args{
				ctx:      context.Background(),
				username: "testuser",
			},
			want:    user,
			wantErr: false,
		},
		{
			name: "user not found by username",
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
						GetUserByUsername(gomock.Any(), "nonexistent").
						Return(nil, sql.ErrNoRows)
				},
			},
			args: args{
				ctx:      context.Background(),
				username: "nonexistent",
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrUserNotFound,
		},
		{
			name: "database error getting user by username",
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
						GetUserByUsername(gomock.Any(), "testuser").
						Return(nil, errors.New("database error"))
				},
			},
			args: args{
				ctx:      context.Background(),
				username: "testuser",
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrUserNotFound,
		},
		{
			name: "empty username",
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
						GetUserByUsername(gomock.Any(), "").
						Return(nil, sql.ErrNoRows)
				},
			},
			args: args{
				ctx:      context.Background(),
				username: "",
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrUserNotFound,
		},
		{
			name: "username with underscores",
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
						GetUserByUsername(gomock.Any(), "test_user_123").
						Return(user, nil)
				},
			},
			args: args{
				ctx:      context.Background(),
				username: "test_user_123",
			},
			want:    user,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockUOW := mockuow.NewMockUnitOfWork(ctrl)
			mockUsersRepo := mockrepositories.NewMockUsersRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockUsersRepository != nil {
				tt.fields.mockUsersRepository(mockUsersRepo)
			}

			newUsersRepoFunc := func(_ pg.Transaction) interfaces.UsersRepository {
				return mockUsersRepo
			}

			s := service.New(mockUOW, newUsersRepoFunc)

			// Act
			got, err := s.GetUserByUsername(tt.args.ctx, tt.args.username)

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

func TestService_UpdateUser(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW             func(*mockuow.MockUnitOfWork)
		mockUsersRepository func(*mockrepositories.MockUsersRepository)
	}

	type args struct {
		ctx  context.Context
		user domains.User
	}

	updatedUser := &domains.User{
		ID:       1,
		Username: "newusername",
		Email:    "user@example.com",
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
			name: "successfully update user",
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
						UpdateUser(gomock.Any(), gomock.Any()).
						Return(nil)

					ur.EXPECT().
						GetUserByID(gomock.Any(), uint64(1)).
						Return(updatedUser, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				user: domains.User{
					ID:       1,
					Username: "newusername",
					Email:    "user@example.com",
				},
			},
			want:    updatedUser,
			wantErr: false,
		},
		{
			name: "error updating user",
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
						UpdateUser(gomock.Any(), gomock.Any()).
						Return(errors.New("database error"))
				},
			},
			args: args{
				ctx: context.Background(),
				user: domains.User{
					ID:       1,
					Username: "newusername",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "error getting updated user",
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
						UpdateUser(gomock.Any(), gomock.Any()).
						Return(nil)

					ur.EXPECT().
						GetUserByID(gomock.Any(), uint64(1)).
						Return(nil, errors.New("user not found"))
				},
			},
			args: args{
				ctx: context.Background(),
				user: domains.User{
					ID:       1,
					Username: "newusername",
				},
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
			mockUsersRepo := mockrepositories.NewMockUsersRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockUsersRepository != nil {
				tt.fields.mockUsersRepository(mockUsersRepo)
			}

			newUsersRepoFunc := func(_ pg.Transaction) interfaces.UsersRepository {
				return mockUsersRepo
			}

			s := service.New(mockUOW, newUsersRepoFunc)

			// Act
			got, err := s.UpdateUser(tt.args.ctx, tt.args.user)

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
