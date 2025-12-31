package usecases

import (
	"context"
	"fmt"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/security"
	"github.com/DKhorkov/libs/validation"
)

type UsersUseCases struct {
	usersService     interfaces.UsersService
	securityConfig   security.Config
	validationConfig config.ValidationConfig
}

func NewUsersUseCases(
	usersService interfaces.UsersService,
	securityConfig security.Config,
	validationConfig config.ValidationConfig,
) *UsersUseCases {
	return &UsersUseCases{
		usersService:     usersService,
		securityConfig:   securityConfig,
		validationConfig: validationConfig,
	}
}

func (u *UsersUseCases) GetUsers(
	ctx context.Context,
	filters *domains.UsersFilters,
	pagination *domains.Pagination,
) ([]domains.User, error) {
	return u.usersService.GetUsers(ctx, filters, pagination)
}

func (u *UsersUseCases) GetUserByID(ctx context.Context, id uint64) (*domains.User, error) {
	return u.usersService.GetUserByID(ctx, id)
}

func (u *UsersUseCases) UpdateUser(
	ctx context.Context,
	userData domains.UpdateUserDTO,
) (*domains.User, error) {
	if !validation.ValidateValueByRules(userData.Username, u.validationConfig.UsernameRegExps) {
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
