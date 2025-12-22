package usecases

import (
	"context"
	"github.com/DKhorkov/khodfeltschat/internal/config"
	"github.com/DKhorkov/khodfeltschat/internal/domains"
	"github.com/DKhorkov/khodfeltschat/internal/interfaces"
	"github.com/DKhorkov/libs/cache"
	"github.com/DKhorkov/libs/security"
	"github.com/DKhorkov/libs/validation"
)

func NewUsersUseCases(
	usersService interfaces.UsersService,
	securityConfig security.Config,
	validationConfig config.ValidationConfig,
	cacheProvider cache.Provider,
) *UsersUseCases {
	return &UsersUseCases{
		usersService:     usersService,
		securityConfig:   securityConfig,
		validationConfig: validationConfig,
		cacheProvider:    cacheProvider,
	}
}

type UsersUseCases struct {
	usersService     interfaces.UsersService
	securityConfig   security.Config
	validationConfig config.ValidationConfig
	cacheProvider    cache.Provider
}

func (u *UsersUseCases) GetUsers(
	ctx context.Context,
	filters *domains.UsersFilters,
	pagination *domains.Pagination,
) ([]domains.User, error) {
	return u.usersService.GetUsers(ctx, filters, pagination)
}

func (u *UsersUseCases) UpdateUser(
	ctx context.Context,
	userData domains.RawUpdateUserDTO,
) (*domains.User, error) {
	if userData.Username != nil && !validation.ValidateValueByRules(
		*userData.Username,
		u.validationConfig.UsernameRegExps,
	) {
		return nil, &validation.Error{Message: "invalid username"}
	}

	accessTokenPayload, err := security.ParseJWT(
		userData.AccessToken,
		u.securityConfig.JWT.SecretKey,
	)
	if err != nil {
		return nil, &security.InvalidJWTError{}
	}

	floatUserID, ok := accessTokenPayload.(float64)
	if !ok {
		return nil, &security.InvalidJWTError{}
	}

	userID := uint64(floatUserID)

	user, err := u.usersService.GetUserByID(ctx, userID)
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

func (u *UsersUseCases) GetMe(ctx context.Context, accessToken string) (*domains.User, error) {
	accessTokenPayload, err := security.ParseJWT(accessToken, u.securityConfig.JWT.SecretKey)
	if err != nil {
		return nil, &security.InvalidJWTError{}
	}

	floatUserID, ok := accessTokenPayload.(float64)
	if !ok {
		return nil, &security.InvalidJWTError{}
	}

	userID := uint64(floatUserID)

	return u.usersService.GetUserByID(ctx, userID)
}
