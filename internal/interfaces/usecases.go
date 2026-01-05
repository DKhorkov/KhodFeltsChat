package interfaces

import (
	"context"

	"github.com/DKhorkov/kfc/internal/domains"
)

//go:generate mockgen -source=usecases.go -destination=../../mocks/usecases/users_usecases.go -package=mockusecases -exclude_interfaces=AuthUseCases,ChatsUseCases
type UsersUseCases interface {
	GetUsers(
		ctx context.Context,
		filters *domains.UsersFilters,
		pagination *domains.Pagination,
	) ([]domains.User, error)
	GetUserByID(ctx context.Context, id uint64) (*domains.User, error)
	UpdateUser(ctx context.Context, userData domains.UpdateUserDTO) (*domains.User, error)
}

//go:generate mockgen -source=usecases.go -destination=../../mocks/usecases/auth_usecases.go -package=mockusecases -exclude_interfaces=UsersUseCases,ChatsUseCases
type AuthUseCases interface {
	RegisterUser(ctx context.Context, dto domains.RegisterDTO) (*domains.User, error)
	LoginUser(ctx context.Context, dto domains.LoginDTO) (*domains.TokensDTO, error)
	LogoutUser(ctx context.Context, userID uint64) error
	RefreshTokens(ctx context.Context, refreshToken string) (*domains.TokensDTO, error)
	VerifyEmail(ctx context.Context, verifyEmailToken string) error
	ForgetPassword(ctx context.Context, forgetPasswordToken, newPassword string) error
	SendForgetPasswordMessage(ctx context.Context, email string) error
	ChangePassword(ctx context.Context, dto domains.ChangePasswordDTO) error
	SendVerifyEmailMessage(ctx context.Context, email string) error
}

//go:generate mockgen -source=usecases.go -destination=../../mocks/usecases/chats_usecases.go -package=mockusecases -exclude_interfaces=UsersUseCases,AuthUseCases
type ChatsUseCases interface {
	GetChatMembers(chatID string) ([]domains.User, error)
}
