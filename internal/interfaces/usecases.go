package interfaces

import (
	"context"
	"io"

	"github.com/DKhorkov/kfc/internal/domains"
)

//go:generate mockgen -source=usecases.go -destination=../../mocks/usecases/users_usecases.go -package=mockusecases -exclude_interfaces=AuthUseCases,ChatsUseCases,MessagesUseCases,NotificationsUseCases,SettingsUseCases,WebPushSubscriptionsUseCases,FileStorageUseCases
type UsersUseCases interface {
	GetUsers(
		ctx context.Context,
		filters *domains.UsersFilters,
		pagination *domains.Pagination,
	) ([]domains.User, error)
	GetUserByID(ctx context.Context, id uint64) (*domains.User, error)
	UpdateUser(ctx context.Context, userData domains.UpdateUserDTO) (*domains.User, error)
	UpdateAvatar(ctx context.Context, userID uint64, data io.Reader) (string, error)
	DeleteAvatar(ctx context.Context, userID uint64) error
}

//go:generate mockgen -source=usecases.go -destination=../../mocks/usecases/auth_usecases.go -package=mockusecases -exclude_interfaces=UsersUseCases,ChatsUseCases,MessagesUseCases,NotificationsUseCases,SettingsUseCases,WebPushSubscriptionsUseCases,FileStorageUseCases
type AuthUseCases interface {
	RegisterUser(ctx context.Context, dto domains.RegisterDTO) (*domains.User, error)
	LoginUser(ctx context.Context, dto domains.LoginDTO) (*domains.TokensDTO, error)
	LogoutUser(ctx context.Context, refreshToken string) error
	LogoutUserFromAllSessions(ctx context.Context, userID uint64) error
	RefreshTokens(ctx context.Context, refreshToken string) (*domains.TokensDTO, error)
	VerifyEmail(ctx context.Context, verifyEmailToken string) error
	ForgetPassword(ctx context.Context, forgetPasswordToken, newPassword string) error
	SendForgetPasswordMessage(ctx context.Context, email string) error
	ChangePassword(ctx context.Context, dto domains.ChangePasswordDTO) error
	SendVerifyEmailMessage(ctx context.Context, email string) error
}

//go:generate mockgen -source=usecases.go -destination=../../mocks/usecases/chats_usecases.go -package=mockusecases -exclude_interfaces=UsersUseCases,AuthUseCases,MessagesUseCases,NotificationsUseCases,SettingsUseCases,WebPushSubscriptionsUseCases,FileStorageUseCases
type ChatsUseCases interface {
	GetChatMembers(ctx context.Context, chatID uint64) ([]domains.User, error)
	GetUserChats(
		ctx context.Context,
		userID uint64,
		pagination *domains.Pagination,
	) ([]domains.Chat, error)
	CreateChat(ctx context.Context, chat domains.Chat) (*domains.Chat, error)
}

//go:generate mockgen -source=usecases.go -destination=../../mocks/usecases/messages_usecases.go -package=mockusecases -exclude_interfaces=UsersUseCases,AuthUseCases,ChatsUseCases,NotificationsUseCases,SettingsUseCases,WebPushSubscriptionsUseCases,FileStorageUseCases
type MessagesUseCases interface {
	MessagesService
}

//go:generate mockgen -source=usecases.go -destination=../../mocks/usecases/notifications_usecases.go -package=mockusecases -exclude_interfaces=UsersUseCases,ChatsUseCases,MessagesUseCases,AuthUseCases,SettingsUseCases,WebPushSubscriptionsUseCases,FileStorageUseCases
type NotificationsUseCases interface {
	SendVerifyEmailMessage(ctx context.Context, userID uint64) error
	SendForgetPasswordMessage(ctx context.Context, userID uint64) error
	SendNewMessageByEmail(
		ctx context.Context,
		userID uint64,
		payload domains.NewMessagePayload,
	) error
	SendNewMessageByWebPush(
		ctx context.Context,
		userID uint64,
		payload domains.NewMessagePayload,
	) error
}

//go:generate mockgen -source=usecases.go -destination=../../mocks/usecases/settings_usecases.go -package=mockusecases -exclude_interfaces=UsersUseCases,AuthUseCases,ChatsUseCases,MessagesUseCases,NotificationsUseCases,WebPushSubscriptionsUseCases,FileStorageUseCases
type SettingsUseCases interface {
	GetSettingsByUserID(ctx context.Context, userID uint64) (*domains.Settings, error)
	UpdateSettings(ctx context.Context, settings domains.Settings) (*domains.Settings, error)
}

//go:generate mockgen -source=usecases.go -destination=../../mocks/usecases/web_push_subscriptions_usecases.go -package=mockusecases -exclude_interfaces=UsersUseCases,AuthUseCases,ChatsUseCases,MessagesUseCases,NotificationsUseCases,SettingsUseCases,FileStorageUseCases
type WebPushSubscriptionsUseCases interface {
	CreateWebPushSubscription(
		ctx context.Context,
		subscription domains.WebPushSubscription,
	) (*domains.WebPushSubscription, error)
	DeleteWebPushSubscription(ctx context.Context, id uint64) error
}

//go:generate mockgen -source=usecases.go -destination=../../mocks/usecases/file_storage_usecases.go -package=mockusecases -exclude_interfaces=UsersUseCases,AuthUseCases,ChatsUseCases,MessagesUseCases,NotificationsUseCases,SettingsUseCases,WebPushSubscriptionsUseCases
type FileStorageUseCases interface {
	Upload(ctx context.Context, path string, data io.Reader) (string, error)
	Download(ctx context.Context, path string) ([]byte, error)
	Delete(ctx context.Context, path string) error
}
