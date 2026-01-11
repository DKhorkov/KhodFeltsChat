package interfaces

import (
	"context"

	"github.com/DKhorkov/kfc/internal/domains"
)

//go:generate mockgen -source=usecases.go -destination=../../mocks/usecases/users_usecases.go -package=mockusecases -exclude_interfaces=AuthUseCases,ChatsUseCases,MessagesUseCases
type UsersUseCases interface {
	GetUsers(
		ctx context.Context,
		filters *domains.UsersFilters,
		pagination *domains.Pagination,
	) ([]domains.User, error)
	GetUserByID(ctx context.Context, id uint64) (*domains.User, error)
	UpdateUser(ctx context.Context, userData domains.UpdateUserDTO) (*domains.User, error)
}

//go:generate mockgen -source=usecases.go -destination=../../mocks/usecases/auth_usecases.go -package=mockusecases -exclude_interfaces=UsersUseCases,ChatsUseCases,MessagesUseCases
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

//go:generate mockgen -source=usecases.go -destination=../../mocks/usecases/chats_usecases.go -package=mockusecases -exclude_interfaces=UsersUseCases,AuthUseCases,MessagesUseCases
type ChatsUseCases interface {
	GetChatMembers(ctx context.Context, chatID uint64) ([]domains.User, error)
	GetUserChats(
		ctx context.Context,
		userID uint64,
		pagination *domains.Pagination,
	) ([]domains.Chat, error)
	CreateChat(ctx context.Context, chat domains.Chat) (*domains.Chat, error)
}

//go:generate mockgen -source=usecases.go -destination=../../mocks/usecases/messages_usecases.go -package=mockusecases -exclude_interfaces=UsersUseCases,AuthUseCases,ChatsUseCases
type MessagesUseCases interface {
	MessagesService
}
