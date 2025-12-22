package interfaces

import (
	"context"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
)

//go:generate mockgen -source=repositories.go -destination=../../mocks/repositories/emails_repository.go -package=mockrepositories -exclude_interfaces=AuthRepository,UsersRepository
type EmailsRepository interface {
	SendVerifyEmailMessage(ctx context.Context, user domains.User) error
	SendForgetPasswordMessage(ctx context.Context, user domains.User) error
}

//go:generate mockgen -source=repositories.go -destination=../../mocks/repositories/users_repository.go -package=mockrepositories -exclude_interfaces=AuthRepository,EmailsRepository
type UsersRepository interface {
	GetUserByID(ctx context.Context, id uint64) (*domains.User, error)
	GetUsers(ctx context.Context, filters *domains.UsersFilters, pagination *domains.Pagination) ([]domains.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domains.User, error)
	GetUserByUsername(ctx context.Context, username string) (*domains.User, error)
	UpdateUser(ctx context.Context, userProfileData domains.UpdateUserDTO) error
}

//go:generate mockgen -source=repositories.go -destination=../../mocks/repositories/auth_repository.go -package=mockrepositories -exclude_interfaces=UsersRepository,EmailsRepository
type AuthRepository interface {
	RegisterUser(ctx context.Context, userData domains.RegisterDTO) (userID uint64, err error)
	CreateRefreshToken(
		ctx context.Context,
		userID uint64,
		value string,
		ttl time.Duration,
	) (refreshTokenID uint64, err error)
	GetRefreshTokenByUserID(ctx context.Context, userID uint64) (*domains.RefreshToken, error)
	ExpireRefreshToken(ctx context.Context, refreshToken string) error
	VerifyEmail(ctx context.Context, userID uint64) error
	ChangePassword(ctx context.Context, userID uint64, newPassword string) error
}
