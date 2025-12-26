package interfaces

import (
	"context"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
)

//go:generate mockgen -source=services.go -destination=../../mocks/services/users_service.go -package=mockservices -exclude_interfaces=AuthService
type UsersService interface {
	GetUserByID(ctx context.Context, id uint64) (*domains.User, error)
	GetUsers(
		ctx context.Context,
		filters *domains.UsersFilters,
		pagination *domains.Pagination,
	) ([]domains.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domains.User, error)
	GetUserByUsername(ctx context.Context, username string) (*domains.User, error)
	UpdateUser(ctx context.Context, userProfileData domains.UpdateUserDTO) (*domains.User, error)
}

//go:generate mockgen -source=services.go -destination=../../mocks/services/auth_service.go -package=mockservices -exclude_interfaces=UsersService
type AuthService interface {
	RegisterUser(ctx context.Context, userData domains.RegisterDTO) (*domains.User, error)
	CreateRefreshToken(
		ctx context.Context,
		userID uint64,
		value string,
		ttl time.Duration,
	) (*domains.RefreshToken, error)
	GetRefreshTokenByUserID(ctx context.Context, userID uint64) (*domains.RefreshToken, error)
	ExpireRefreshToken(ctx context.Context, refreshToken string) error
	VerifyEmail(ctx context.Context, userID uint64) error
	ForgetPassword(ctx context.Context, userID uint64, newPassword string) error
	ChangePassword(ctx context.Context, userID uint64, newPassword string) error
	SendForgetPasswordMessage(ctx context.Context, email string) error
	SendVerifyEmailMessage(ctx context.Context, email string) error
}
