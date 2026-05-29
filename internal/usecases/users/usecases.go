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

const (
	avatarFileExtension = ".jpg"
	imageWidth          = 256
	imageHeight         = 256
)

type UseCases struct {
	usersService       interfaces.UsersService
	fileStorageService interfaces.FileStorageService
	securityConfig     security.Config
	validationConfig   config.ValidationConfig
	fileStorageConfig  config.FileStorageConfig
}

func New(
	usersService interfaces.UsersService,
	fileStorageService interfaces.FileStorageService,
	securityConfig security.Config,
	validationConfig config.ValidationConfig,
	fileStorageConfig config.FileStorageConfig,
) *UseCases {
	return &UseCases{
		usersService:       usersService,
		fileStorageService: fileStorageService,
		securityConfig:     securityConfig,
		validationConfig:   validationConfig,
		fileStorageConfig:  fileStorageConfig,
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

	if userData.Username != nil {
		user.Username = *userData.Username
	}

	return u.usersService.UpdateUser(ctx, *user)
}

func (u *UseCases) UpdateAvatar(
	ctx context.Context,
	userID uint64,
	data io.Reader,
) (string, error) {
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

	resized := common.ResizeImage(img, imageWidth, imageHeight)

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
			if err = u.fileStorageService.Delete(ctx, oldUUID+avatarFileExtension); err != nil {
				return "", err
			}
		}
	}

	fileUUID := uuid.New().String()
	fileName := fileUUID + avatarFileExtension

	if err = u.fileStorageService.Upload(ctx, fileName, bytes.NewReader(buf.Bytes())); err != nil {
		return "", err
	}

	avatarURL := u.fileStorageConfig.BaseDownloadURL + "/" + fileName

	user.AvatarPath = &avatarURL

	if _, err = u.usersService.UpdateUser(ctx, *user); err != nil {
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
		if err = u.fileStorageService.Delete(ctx, oldUUID+avatarFileExtension); err != nil {
			return err
		}
	}

	user.AvatarPath = nil

	_, err = u.usersService.UpdateUser(ctx, *user)

	return err
}
