package users

import (
	"bytes"
	"context"
	"fmt"
	"image/jpeg"
	"io"

	"github.com/DKhorkov/kfc/internal/common"
	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/security"
	"github.com/DKhorkov/libs/validation"
	"github.com/google/uuid"
)

type UseCases struct {
	usersService        interfaces.UsersService
	fileStorageUseCases interfaces.FileStorageUseCases
	securityConfig      security.Config
	validationConfig    config.ValidationConfig
	fileStorageConfig   config.FileStorageConfig
}

func New(
	usersService interfaces.UsersService,
	fileStorageUseCases interfaces.FileStorageUseCases,
	securityConfig security.Config,
	validationConfig config.ValidationConfig,
	fileStorageConfig config.FileStorageConfig,
) *UseCases {
	return &UseCases{
		usersService:        usersService,
		fileStorageUseCases: fileStorageUseCases,
		securityConfig:      securityConfig,
		validationConfig:    validationConfig,
		fileStorageConfig:   fileStorageConfig,
	}
}

func (u *UseCases) GetUsers(
	ctx context.Context,
	filters *domains.UsersFilters,
	pagination *domains.Pagination,
) ([]domains.User, error) {
	return u.usersService.GetUsers(ctx, filters, pagination)
}

func (u *UseCases) GetUserByID(ctx context.Context, id uint64) (*domains.User, error) {
	return u.usersService.GetUserByID(ctx, id)
}

func (u *UseCases) UpdateUser(
	ctx context.Context,
	userData domains.UpdateUserDTO,
) (*domains.User, error) {
	if userData.Username != nil &&
		!validation.ValidateValueByRules(*userData.Username, u.validationConfig.UsernameRegExps) {
		return nil, fmt.Errorf("%w: invalid username", customerrors.ErrValidationFailed)
	}

	user, err := u.usersService.GetUserByID(ctx, userData.ID)
	if err != nil {
		return nil, err
	}

	return u.usersService.UpdateUser(
		ctx,
		domains.UpdateUserDTO{
			ID:       user.ID,
			Username: userData.Username,
		},
	)
}

func (u *UseCases) UpdateAvatar(ctx context.Context, userID uint64, data io.Reader) (string, error) {
	rawData, err := io.ReadAll(io.LimitReader(data, int64(u.fileStorageConfig.MaxSize)+1))
	if err != nil {
		return "", err
	}

	if len(rawData) > u.fileStorageConfig.MaxSize {
		return "", customerrors.ErrFileTooLarge
	}

	img, err := common.DecodeImage(rawData)
	if err != nil {
		return "", fmt.Errorf("%w: %w", customerrors.ErrInvalidImageFormat, err)
	}

	resized := common.ResizeImage(img, 256, 256)

	var buf bytes.Buffer
	if err = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 85}); err != nil {
		return "", err
	}

	user, err := u.usersService.GetUserByID(ctx, userID)
	if err != nil {
		return "", err
	}

	if user.AvatarPath != nil {
		oldUUID := common.ExtractUUIDFromURL(*user.AvatarPath)
		if oldUUID != "" {
			_ = u.fileStorageUseCases.Delete(ctx, oldUUID+".jpg")
		}
	}

	fileUUID := uuid.New().String()
	fileName := fileUUID + ".jpg"

	avatarURL, err := u.fileStorageUseCases.Upload(ctx, fileName, bytes.NewReader(buf.Bytes()))
	if err != nil {
		return "", err
	}

	if _, err = u.usersService.UpdateUser(ctx, domains.UpdateUserDTO{
		ID:         userID,
		AvatarPath: &avatarURL,
	}); err != nil {
		return "", err
	}

	return avatarURL, nil
}

func (u *UseCases) DeleteAvatar(ctx context.Context, userID uint64) error {
	user, err := u.usersService.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.AvatarPath == nil {
		return nil
	}

	oldUUID := common.ExtractUUIDFromURL(*user.AvatarPath)
	if oldUUID != "" {
		if err = u.fileStorageUseCases.Delete(ctx, oldUUID+".jpg"); err != nil {
			return err
		}
	}

	emptyPath := ""

	_, err = u.usersService.UpdateUser(ctx, domains.UpdateUserDTO{
		ID:         userID,
		AvatarPath: &emptyPath,
	})

	return err
}
