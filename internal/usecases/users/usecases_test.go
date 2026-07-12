package users_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
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

// buildJPEG — маленький валидный JPEG для UpdateAvatar тестов.
func buildJPEG(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 50}); err != nil {
		t.Fatalf("build test jpeg: %v", err)
	}

	return buf.Bytes()
}

func newAvatarUseCase(t *testing.T) (
	*users.UseCases,
	*mockservices.MockUsersService,
	*mockservices.MockFileStorageService,
) {
	t.Helper()

	ctrl := gomock.NewController(t)
	usersSvc := mockservices.NewMockUsersService(ctrl)
	fileSvc := mockservices.NewMockFileStorageService(ctrl)

	uc := users.New(
		usersSvc,
		fileSvc,
		security.Config{HashCost: 10},
		config.ValidationConfig{},
		config.FileStorageConfig{
			MaxSize:         1 * 1024 * 1024, // 1 MB достаточно для тестового JPEG
			BaseDownloadURL: "https://example.com/files",
		},
	)

	return uc, usersSvc, fileSvc
}

func TestUseCases_UpdateAvatar_HappyPath_NoExistingAvatar(t *testing.T) {
	t.Parallel()

	uc, usersSvc, fileSvc := newAvatarUseCase(t)
	data := buildJPEG(t)

	usersSvc.EXPECT().
		GetUserByID(gomock.Any(), uint64(1)).
		Return(&domains.User{ID: 1, AvatarPath: nil}, nil)
	fileSvc.EXPECT().
		Upload(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
	usersSvc.EXPECT().
		UpdateUser(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, u domains.User) (*domains.User, error) {
			assert.NotNil(t, u.AvatarPath)
			assert.Contains(t, *u.AvatarPath, "https://example.com/files/")
			return &u, nil
		})

	url, err := uc.UpdateAvatar(context.Background(), 1, bytes.NewReader(data))
	assert.NoError(t, err)
	assert.Contains(t, url, "https://example.com/files/")
	assert.Contains(t, url, ".jpg")
}

func TestUseCases_UpdateAvatar_HappyPath_DeletesOldAvatar(t *testing.T) {
	t.Parallel()

	uc, usersSvc, fileSvc := newAvatarUseCase(t)
	data := buildJPEG(t)
	oldPath := "https://example.com/files/old-uuid-1234.jpg"

	usersSvc.EXPECT().
		GetUserByID(gomock.Any(), uint64(1)).
		Return(&domains.User{ID: 1, AvatarPath: &oldPath}, nil)
	fileSvc.EXPECT().
		Delete(gomock.Any(), "old-uuid-1234.jpg").
		Return(nil)
	fileSvc.EXPECT().
		Upload(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
	usersSvc.EXPECT().
		UpdateUser(gomock.Any(), gomock.Any()).
		Return(&domains.User{ID: 1}, nil)

	_, err := uc.UpdateAvatar(context.Background(), 1, bytes.NewReader(data))
	assert.NoError(t, err)
}

func TestUseCases_UpdateAvatar_FileTooLarge(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	usersSvc := mockservices.NewMockUsersService(ctrl)
	fileSvc := mockservices.NewMockFileStorageService(ctrl)

	uc := users.New(
		usersSvc, fileSvc,
		security.Config{},
		config.ValidationConfig{},
		config.FileStorageConfig{MaxSize: 10}, // 10 байт лимит
	)

	// io.LimitReader читает MaxSize+1 = 11 байт, > 10 → ErrFileTooLarge.
	data := bytes.Repeat([]byte("A"), 100)

	_, err := uc.UpdateAvatar(context.Background(), 1, bytes.NewReader(data))
	assert.ErrorIs(t, err, customerrors.ErrFileTooLarge)
}

func TestUseCases_UpdateAvatar_InvalidImageFormat(t *testing.T) {
	t.Parallel()

	uc, _, _ := newAvatarUseCase(t)

	// Не-JPEG байты → DecodeImage не распознаёт → ErrInvalidImageFormat.
	_, err := uc.UpdateAvatar(context.Background(), 1, bytes.NewReader([]byte("not an image")))
	assert.ErrorIs(t, err, customerrors.ErrInvalidImageFormat)
}

func TestUseCases_UpdateAvatar_GetUserByIDError(t *testing.T) {
	t.Parallel()

	uc, usersSvc, _ := newAvatarUseCase(t)
	data := buildJPEG(t)

	usersSvc.EXPECT().
		GetUserByID(gomock.Any(), uint64(1)).
		Return(nil, errors.New("user not found"))

	_, err := uc.UpdateAvatar(context.Background(), 1, bytes.NewReader(data))
	assert.Error(t, err)
}

func TestUseCases_UpdateAvatar_DeleteOldAvatarError(t *testing.T) {
	t.Parallel()

	uc, usersSvc, fileSvc := newAvatarUseCase(t)
	data := buildJPEG(t)
	oldPath := "https://example.com/files/old.jpg"

	usersSvc.EXPECT().
		GetUserByID(gomock.Any(), uint64(1)).
		Return(&domains.User{ID: 1, AvatarPath: &oldPath}, nil)
	fileSvc.EXPECT().
		Delete(gomock.Any(), "old.jpg").
		Return(errors.New("delete failed"))

	_, err := uc.UpdateAvatar(context.Background(), 1, bytes.NewReader(data))
	assert.Error(t, err)
}

func TestUseCases_UpdateAvatar_UploadError(t *testing.T) {
	t.Parallel()

	uc, usersSvc, fileSvc := newAvatarUseCase(t)
	data := buildJPEG(t)

	usersSvc.EXPECT().
		GetUserByID(gomock.Any(), uint64(1)).
		Return(&domains.User{ID: 1, AvatarPath: nil}, nil)
	fileSvc.EXPECT().
		Upload(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errors.New("upload failed"))

	_, err := uc.UpdateAvatar(context.Background(), 1, bytes.NewReader(data))
	assert.Error(t, err)
}

func TestUseCases_UpdateAvatar_UpdateUserError(t *testing.T) {
	t.Parallel()

	uc, usersSvc, fileSvc := newAvatarUseCase(t)
	data := buildJPEG(t)

	usersSvc.EXPECT().
		GetUserByID(gomock.Any(), uint64(1)).
		Return(&domains.User{ID: 1, AvatarPath: nil}, nil)
	fileSvc.EXPECT().
		Upload(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
	usersSvc.EXPECT().
		UpdateUser(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("update failed"))

	_, err := uc.UpdateAvatar(context.Background(), 1, bytes.NewReader(data))
	assert.Error(t, err)
}

func TestUseCases_DeleteAvatar_HappyPath(t *testing.T) {
	t.Parallel()

	uc, usersSvc, fileSvc := newAvatarUseCase(t)
	avatarPath := "https://example.com/files/some-uuid.jpg"

	usersSvc.EXPECT().
		GetUserByID(gomock.Any(), uint64(1)).
		Return(&domains.User{ID: 1, AvatarPath: &avatarPath}, nil)
	fileSvc.EXPECT().
		Delete(gomock.Any(), "some-uuid.jpg").
		Return(nil)
	usersSvc.EXPECT().
		UpdateUser(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, u domains.User) (*domains.User, error) {
			assert.Nil(t, u.AvatarPath)
			return &u, nil
		})

	assert.NoError(t, uc.DeleteAvatar(context.Background(), 1))
}

func TestUseCases_DeleteAvatar_NoAvatar_EarlyReturn(t *testing.T) {
	t.Parallel()

	uc, usersSvc, _ := newAvatarUseCase(t)

	usersSvc.EXPECT().
		GetUserByID(gomock.Any(), uint64(1)).
		Return(&domains.User{ID: 1, AvatarPath: nil}, nil)
	// Delete и UpdateUser НЕ вызываются.

	assert.NoError(t, uc.DeleteAvatar(context.Background(), 1))
}

func TestUseCases_DeleteAvatar_GetUserByIDError(t *testing.T) {
	t.Parallel()

	uc, usersSvc, _ := newAvatarUseCase(t)

	usersSvc.EXPECT().
		GetUserByID(gomock.Any(), uint64(1)).
		Return(nil, errors.New("not found"))

	assert.Error(t, uc.DeleteAvatar(context.Background(), 1))
}

func TestUseCases_DeleteAvatar_DeleteError(t *testing.T) {
	t.Parallel()

	uc, usersSvc, fileSvc := newAvatarUseCase(t)
	avatarPath := "https://example.com/files/some-uuid.jpg"

	usersSvc.EXPECT().
		GetUserByID(gomock.Any(), uint64(1)).
		Return(&domains.User{ID: 1, AvatarPath: &avatarPath}, nil)
	fileSvc.EXPECT().
		Delete(gomock.Any(), "some-uuid.jpg").
		Return(errors.New("delete failed"))

	assert.Error(t, uc.DeleteAvatar(context.Background(), 1))
}

func TestUseCases_DeleteAvatar_UpdateUserError(t *testing.T) {
	t.Parallel()

	uc, usersSvc, fileSvc := newAvatarUseCase(t)
	avatarPath := "https://example.com/files/some-uuid.jpg"

	usersSvc.EXPECT().
		GetUserByID(gomock.Any(), uint64(1)).
		Return(&domains.User{ID: 1, AvatarPath: &avatarPath}, nil)
	fileSvc.EXPECT().
		Delete(gomock.Any(), "some-uuid.jpg").
		Return(nil)
	usersSvc.EXPECT().
		UpdateUser(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("update failed"))

	assert.Error(t, uc.DeleteAvatar(context.Background(), 1))
}

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

			uc := users.New(
				mockUsersService,
				nil,
				securityConfig,
				validationConfig,
				config.FileStorageConfig{},
			)

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

			uc := users.New(
				mockUsersService,
				nil,
				securityConfig,
				validationConfig,
				config.FileStorageConfig{},
			)

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
						UpdateUser(gomock.Any(), domains.User{
							ID:       1,
							Username: "newusername",
							Email:    "user@example.com",
						}).
						Return(updatedUser, nil)
				},
				validationRules: []string{"^[a-zA-Z0-9_]{3,20}$"},
			},
			args: args{
				ctx: context.Background(),
				userData: domains.UpdateUserDTO{
					ID:       1,
					Username: pointers.New("newusername"),
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
					Username: pointers.New("newusername"),
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
					Username: pointers.New("newusername"),
				},
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("update failed"),
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
					Username: pointers.New("thisusernameistoolongforvalidation"),
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
						UpdateUser(gomock.Any(), domains.User{
							ID:       1,
							Username: "oldusername",
							Email:    "user@example.com",
						}).
						Return(existingUser, nil)
				},
				validationRules: []string{"^[a-zA-Z0-9_]{3,20}$"},
			},
			args: args{
				ctx: context.Background(),
				userData: domains.UpdateUserDTO{
					ID:       1,
					Username: pointers.New("oldusername"),
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

			uc := users.New(
				mockUsersService,
				nil,
				securityConfig,
				validationConfig,
				config.FileStorageConfig{},
			)

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
