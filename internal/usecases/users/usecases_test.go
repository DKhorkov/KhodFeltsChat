package users_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/usecases/users"
	mockservices "github.com/DKhorkov/kfc/mocks/services"
	"github.com/DKhorkov/libs/pointers"
	"github.com/DKhorkov/libs/security"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestUseCases_GetUsers(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUsersService func(*mockservices.MockUsersService)
	}

	type args struct {
		ctx        context.Context
		filters    *domains.UsersFilters
		pagination *domains.Pagination
	}

	expectedUsers := []domains.User{
		{ID: 1, Username: "user1", Email: "user1@test.com"},
		{ID: 2, Username: "user2", Email: "user2@test.com"},
	}

	limit := uint64(10)
	offset := uint64(0)
	pagination := &domains.Pagination{Limit: &limit, Offset: &offset}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []domains.User
		wantErr bool
		err     error
	}{
		{
			name: "successfully get users without filters",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUsers(gomock.Any(), (*domains.UsersFilters)(nil), pagination).
						Return(expectedUsers, nil)
				},
			},
			args: args{
				ctx:        context.Background(),
				filters:    nil,
				pagination: pagination,
			},
			want:    expectedUsers,
			wantErr: false,
		},
		{
			name: "successfully get users with username filter",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					filters := &domains.UsersFilters{Username: pointers.New("test")}
					us.EXPECT().
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
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUsers(gomock.Any(), (*domains.UsersFilters)(nil), (*domains.Pagination)(nil)).
						Return(expectedUsers, nil)
				},
			},
			args: args{
				ctx:        context.Background(),
				filters:    nil,
				pagination: nil,
			},
			want:    expectedUsers,
			wantErr: false,
		},
		{
			name: "service returns error",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
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
		{
			name: "empty user list returned",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockUsersService := mockservices.NewMockUsersService(ctrl)

			if tt.fields.mockUsersService != nil {
				tt.fields.mockUsersService(mockUsersService)
			}

			securityConfig := security.Config{HashCost: 10}
			validationConfig := config.ValidationConfig{
				UsernameRegExps: []string{"^[a-zA-Z0-9_]{3,20}$"},
			}

			uc := users.New(mockUsersService, securityConfig, validationConfig)

			// Act
			got, err := uc.GetUsers(tt.args.ctx, tt.args.filters, tt.args.pagination)

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

func TestUseCases_GetUserByID(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUsersService func(*mockservices.MockUsersService)
	}

	type args struct {
		ctx context.Context
		id  uint64
	}

	expectedUser := &domains.User{
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
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(1)).
						Return(expectedUser, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				id:  1,
			},
			want:    expectedUser,
			wantErr: false,
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
				id:  999,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("user not found"),
		},
		{
			name: "service returns other error",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
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
			err:     errors.New("database connection failed"),
		},
		{
			name: "get user with zero ID",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(0)).
						Return(nil, errors.New("invalid user ID"))
				},
			},
			args: args{
				ctx: context.Background(),
				id:  0,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("invalid user ID"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockUsersService := mockservices.NewMockUsersService(ctrl)

			if tt.fields.mockUsersService != nil {
				tt.fields.mockUsersService(mockUsersService)
			}

			securityConfig := security.Config{HashCost: 10}
			validationConfig := config.ValidationConfig{}

			uc := users.New(mockUsersService, securityConfig, validationConfig)

			// Act
			got, err := uc.GetUserByID(tt.args.ctx, tt.args.id)

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

func TestUseCases_UpdateUser(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUsersService func(*mockservices.MockUsersService)
		validationRules  []string
	}

	type args struct {
		ctx      context.Context
		userData domains.UpdateUserDTO
	}

	existingUser := &domains.User{
		ID:       1,
		Username: "oldusername",
		Email:    "user@example.com",
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
			name: "successfully update username",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(1)).
						Return(existingUser, nil)
					us.EXPECT().
						UpdateUser(gomock.Any(), domains.UpdateUserDTO{
							ID:       1,
							Username: "newusername",
						}).
						Return(updatedUser, nil)
				},
				validationRules: []string{"^[a-zA-Z0-9_]{3,20}$"},
			},
			args: args{
				ctx: context.Background(),
				userData: domains.UpdateUserDTO{
					ID:       1,
					Username: "newusername",
				},
			},
			want:    updatedUser,
			wantErr: false,
		},
		{
			name: "user not found",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(999)).
						Return(nil, errors.New("user not found"))
				},
				validationRules: []string{"^[a-zA-Z0-9_]{3,20}$"},
			},
			args: args{
				ctx: context.Background(),
				userData: domains.UpdateUserDTO{
					ID:       999,
					Username: "newusername",
				},
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("user not found"),
		},
		{
			name: "update service returns error",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(1)).
						Return(existingUser, nil)
					us.EXPECT().
						UpdateUser(gomock.Any(), gomock.Any()).
						Return(nil, errors.New("update failed"))
				},
				validationRules: []string{"^[a-zA-Z0-9_]{3,20}$"},
			},
			args: args{
				ctx: context.Background(),
				userData: domains.UpdateUserDTO{
					ID:       1,
					Username: "newusername",
				},
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("update failed"),
		},
		{
			name: "empty username",
			fields: fields{
				validationRules: []string{"^[a-zA-Z0-9_]{3,20}$"},
			},
			args: args{
				ctx: context.Background(),
				userData: domains.UpdateUserDTO{
					ID:       1,
					Username: "",
				},
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrValidationFailed,
		},
		{
			name: "username too long",
			fields: fields{
				validationRules: []string{"^[a-zA-Z0-9_]{3,20}$"},
			},
			args: args{
				ctx: context.Background(),
				userData: domains.UpdateUserDTO{
					ID:       1,
					Username: "thisusernameistoolongforvalidation",
				},
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrValidationFailed,
		},
		{
			name: "update with same username",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(1)).
						Return(existingUser, nil)
					us.EXPECT().
						UpdateUser(gomock.Any(), domains.UpdateUserDTO{
							ID:       1,
							Username: "oldusername", // То же самое имя
						}).
						Return(existingUser, nil)
				},
				validationRules: []string{"^[a-zA-Z0-9_]{3,20}$"},
			},
			args: args{
				ctx: context.Background(),
				userData: domains.UpdateUserDTO{
					ID:       1,
					Username: "oldusername",
				},
			},
			want:    existingUser,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockUsersService := mockservices.NewMockUsersService(ctrl)

			if tt.fields.mockUsersService != nil {
				tt.fields.mockUsersService(mockUsersService)
			}

			securityConfig := security.Config{HashCost: 10}
			validationConfig := config.ValidationConfig{
				UsernameRegExps: tt.fields.validationRules,
			}

			uc := users.New(mockUsersService, securityConfig, validationConfig)

			// Act
			got, err := uc.UpdateUser(tt.args.ctx, tt.args.userData)

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
