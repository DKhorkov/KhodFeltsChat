package interfaces

import (
	"context"

	"github.com/DKhorkov/kfc/internal/domains"
)

//go:generate mockgen -source=usecases.go -destination=../../mocks/usecases/users_usecases.go -package=mockusecases -exclude_interfaces=AuthUseCases
type UsersUseCases interface {
	GetUsers(ctx context.Context, filters *domains.UsersFilters, pagination *domains.Pagination) ([]domains.User, error)
	GetUserByID(ctx context.Context, id uint64) (*domains.User, error)
	GetMe(ctx context.Context, accessToken string) (*domains.User, error)
	UpdateUser(ctx context.Context, userData domains.RawUpdateUserDTO) (*domains.User, error)
}

//go:generate mockgen -source=usecases.go -destination=../../mocks/usecases/auth_usecases.go -package=mockusecases -exclude_interfaces=UsersUseCases
type AuthUseCases interface {
	RegisterUser(ctx context.Context, userData domains.RegisterDTO) (*domains.User, error)
	LoginUser(ctx context.Context, userData domains.LoginDTO) (*domains.TokensDTO, error)
	LogoutUser(ctx context.Context, accessToken string) error
	RefreshTokens(ctx context.Context, refreshToken string) (*domains.TokensDTO, error)
	VerifyUserEmail(ctx context.Context, verifyEmailToken string) error
	ForgetPassword(ctx context.Context, forgetPasswordToken, newPassword string) error
	SendForgetPasswordMessage(ctx context.Context, email string) error
	ChangePassword(ctx context.Context, accessToken, oldPassword, newPassword string) error
	SendVerifyEmailMessage(ctx context.Context, email string) error
}
