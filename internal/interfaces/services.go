package interfaces

import (
	"context"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
)

//go:generate mockgen -source=services.go -destination=../../mocks/services/users_service.go -package=mockservices -exclude_interfaces=AuthService,ChatsService,MessagesService,NotificationsService
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

//go:generate mockgen -source=services.go -destination=../../mocks/services/auth_service.go -package=mockservices -exclude_interfaces=UsersService,ChatsService,MessagesService,NotificationsService
type AuthService interface {
	RegisterUser(ctx context.Context, userData domains.RegisterDTO) (*domains.User, error)
	CreateRefreshToken(
		ctx context.Context,
		userID uint64,
		value string,
		ttl time.Duration,
	) (*domains.RefreshToken, error)
	GetRefreshTokenByUserID(ctx context.Context, userID uint64) (*domains.RefreshToken, error)
	ExpireRefreshToken(ctx context.Context, refreshTokenID uint64) error
	VerifyEmail(ctx context.Context, userID uint64) error
	ForgetPassword(ctx context.Context, userID uint64, newPassword string) error
	ChangePassword(ctx context.Context, userID uint64, newPassword string) error
	SendForgetPasswordMessage(ctx context.Context, email string) error
	SendVerifyEmailMessage(ctx context.Context, email string) error
}

//go:generate mockgen -source=services.go -destination=../../mocks/services/chats_service.go -package=mockservices -exclude_interfaces=UsersService,AuthService,MessagesService,NotificationsService
type ChatsService interface {
	GetChatMembers(ctx context.Context, chatID uint64) ([]domains.User, error)
	GetUserChats(
		ctx context.Context,
		userID uint64,
		pagination *domains.Pagination,
	) ([]domains.Chat, error)
	CreateChat(ctx context.Context, chat domains.Chat) (*domains.Chat, error)
	PrivateChatExists(ctx context.Context, members []domains.User) (bool, error)
}

//go:generate mockgen -source=services.go -destination=../../mocks/services/messages_service.go -package=mockservices -exclude_interfaces=UsersService,AuthService,ChatsService,NotificationsService
type MessagesService interface {
	SaveMessage(ctx context.Context, message domains.Message) (*domains.Message, error)
	GetChatMessages(
		ctx context.Context,
		userID uint64,
		chatID uint64,
		pagination *domains.Pagination,
	) ([]domains.Message, error)
}

//go:generate mockgen -source=services.go -destination=../../mocks/services/notifications_service.go -package=mockservices -exclude_interfaces=UsersService,ChatsService,MessagesService,AuthService
type NotificationsService interface {
	EmailsRepository
}
