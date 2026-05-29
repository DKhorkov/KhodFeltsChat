package interfaces

import (
	"context"
	"io"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
)

//go:generate mockgen -source=repositories.go -destination=../../mocks/repositories/emails_repository.go -package=mockrepositories -exclude_interfaces=AuthRepository,UsersRepository,MessagesRepository,ChatsRepository,SettingsRepository,WebPushSubscriptionsRepository,WebPushRepository,FileStorageRepository
type EmailsRepository interface {
	SendVerifyEmailMessage(ctx context.Context, user domains.User) error
	SendForgetPasswordMessage(ctx context.Context, user domains.User) error
	SendMessage(
		ctx context.Context,
		recipient domains.User,
		message domains.Message,
		chat domains.Chat,
	) error
}

//go:generate mockgen -source=repositories.go -destination=../../mocks/repositories/users_repository.go -package=mockrepositories -exclude_interfaces=AuthRepository,EmailsRepository,MessagesRepository,ChatsRepository,SettingsRepository,WebPushSubscriptionsRepository,WebPushRepository,FileStorageRepository
type UsersRepository interface {
	GetUserByID(ctx context.Context, id uint64) (*domains.User, error)
	GetUsers(
		ctx context.Context,
		filters *domains.UsersFilters,
		pagination *domains.Pagination,
	) ([]domains.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domains.User, error)
	GetUserByUsername(ctx context.Context, username string) (*domains.User, error)
	UpdateUser(ctx context.Context, user domains.User) error
}

//go:generate mockgen -source=repositories.go -destination=../../mocks/repositories/auth_repository.go -package=mockrepositories -exclude_interfaces=UsersRepository,EmailsRepository,MessagesRepository,ChatsRepository,SettingsRepository,WebPushSubscriptionsRepository,WebPushRepository,FileStorageRepository
type AuthRepository interface {
	RegisterUser(ctx context.Context, userData domains.RegisterDTO) (userID uint64, err error)
	CreateRefreshToken(
		ctx context.Context,
		userID uint64,
		value string,
		ttl time.Duration,
	) (refreshTokenID uint64, err error)
	GetRefreshTokenByValue(ctx context.Context, value string) (*domains.RefreshToken, error)
	ExpireRefreshToken(ctx context.Context, refreshTokenID uint64) error
	ExpireAllUserRefreshTokens(ctx context.Context, userID uint64) error
	VerifyEmail(ctx context.Context, userID uint64) error
	ChangePassword(ctx context.Context, userID uint64, newPassword string) error
}

//go:generate mockgen -source=repositories.go -destination=../../mocks/repositories/chats_repository.go -package=mockrepositories -exclude_interfaces=UsersRepository,AuthRepository,MessagesRepository,EmailsRepository,SettingsRepository,WebPushSubscriptionsRepository,WebPushRepository,FileStorageRepository
type ChatsRepository interface {
	GetChatMembers(ctx context.Context, chatID uint64) ([]domains.User, error)
	GetUserChats(
		ctx context.Context,
		userID uint64,
		pagination *domains.Pagination,
	) ([]domains.Chat, error)
	CreateChat(ctx context.Context, chat domains.Chat) (uint64, error)
	GetChatByID(ctx context.Context, id uint64) (*domains.Chat, error)
	PrivateChatExists(ctx context.Context, members []domains.User) (bool, error)
}

//go:generate mockgen -source=repositories.go -destination=../../mocks/repositories/messages_service.go -package=mockrepositories -exclude_interfaces=UsersRepository,AuthRepository,ChatsRepository,EmailsRepository,SettingsRepository,WebPushSubscriptionsRepository,WebPushRepository,FileStorageRepository
type MessagesRepository interface {
	SaveMessage(ctx context.Context, message domains.Message) (uint64, error)
	GetChatMessages(
		ctx context.Context,
		userID uint64,
		chatID uint64,
		pagination *domains.Pagination,
	) ([]domains.Message, error)
	GetMessageByID(ctx context.Context, userID uint64, messageID uint64) (*domains.Message, error)
	ChangeMessagesIsReadStatus(
		ctx context.Context,
		userID uint64,
		messages []domains.Message,
		isRead bool,
	) error
	ReadAllChatMessages(ctx context.Context, userID uint64, chatID uint64) error
	DeleteMessageForUser(ctx context.Context, userID uint64, messageID uint64) error
	DeleteMessageForAll(ctx context.Context, messageID uint64) error
	UpdateMessage(ctx context.Context, dto domains.UpdateMessageDTO) error
}

//go:generate mockgen -source=repositories.go -destination=../../mocks/repositories/settings_repository.go -package=mockrepositories -exclude_interfaces=AuthRepository,UsersRepository,EmailsRepository,MessagesRepository,ChatsRepository,WebPushSubscriptionsRepository,WebPushRepository,FileStorageRepository
type SettingsRepository interface {
	CreateSettings(ctx context.Context, settings domains.Settings) error
	GetSettingsByUserID(ctx context.Context, userID uint64) (*domains.Settings, error)
	UpdateSettings(ctx context.Context, settings domains.Settings) error
}

//go:generate mockgen -source=repositories.go -destination=../../mocks/repositories/web_push_subscriptions_repository.go -package=mockrepositories -exclude_interfaces=AuthRepository,UsersRepository,EmailsRepository,MessagesRepository,ChatsRepository,SettingsRepository,WebPushRepository,FileStorageRepository
type WebPushSubscriptionsRepository interface {
	CreateWebPushSubscription(
		ctx context.Context,
		subscription domains.WebPushSubscription,
	) (uint64, error)
	GetWebPushSubscriptionsByUserID(
		ctx context.Context,
		userID uint64,
	) ([]domains.WebPushSubscription, error)
	DeleteWebPushSubscription(ctx context.Context, id uint64) error
}

//go:generate mockgen -source=repositories.go -destination=../../mocks/repositories/web_push_repository.go -package=mockrepositories -exclude_interfaces=AuthRepository,UsersRepository,EmailsRepository,MessagesRepository,ChatsRepository,SettingsRepository,WebPushSubscriptionsRepository,FileStorageRepository
type WebPushRepository interface {
	SendNotification(
		ctx context.Context,
		subscription domains.WebPushSubscription,
		message domains.Message,
	) error
}

//go:generate mockgen -source=repositories.go -destination=../../mocks/repositories/file_storage_repository.go -package=mockrepositories -exclude_interfaces=AuthRepository,UsersRepository,EmailsRepository,MessagesRepository,ChatsRepository,SettingsRepository,WebPushSubscriptionsRepository,WebPushRepository
type FileStorageRepository interface {
	Upload(ctx context.Context, path string, data io.Reader) error
	Download(ctx context.Context, path string) ([]byte, error)
	Delete(ctx context.Context, path string) error
}
