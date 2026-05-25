package api

import (
	"fmt"
	"net/http"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/change_password"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/forget_password"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/login"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/logout"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/logout_all"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/refresh_tokens"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/register"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/send_forget_password_message"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/send_verify_email_message"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/verify_email"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/chats/create"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/chats/user_chats"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/messages/chat_messages"
	delete_message "github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/messages/delete"
	get_settings "github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/settings/get"
	update_settings "github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/settings/update"
	me "github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/users/me"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/users/update"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/users/user_by_id"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/users/users"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/web_push/subscribe"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/web_push/unsubscribe"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/web_push/vapid_key"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/ws"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/logging"
	customnats "github.com/DKhorkov/libs/nats"
	"github.com/gorilla/mux"
)

const (
	SessionsURL    = "/sessions"
	AllSessionsURL = SessionsURL + "/all"

	UsersURL                  = "/users"
	MeURL                     = UsersURL + "/me"
	GetUserByIDURL            = UsersURL + "/{%s}"
	PasswordURL               = UsersURL + "/password"
	ChangePasswordURL         = PasswordURL + "/change"
	SendForgetPasswordURL     = PasswordURL + "/forget"
	ForgetPasswordURL         = SendForgetPasswordURL + "/{%s}"
	SendVerifyEmailMessageURL = UsersURL + "/email/verify"
	VerifyEmailURL            = SendVerifyEmailMessageURL + "/{%s}"

	WebsocketURL = "/ws"

	SettingsURL = MeURL + "/settings"

	ChatsURL           = "/chats"
	GetChatMessagesURL = ChatsURL + "/{%s}/messages"

	MessagesURL      = "/messages"
	DeleteMessageURL = MessagesURL + "/{%s}"

	WebPushURL            = "/web-push"
	WebPushSubscribeURL   = WebPushURL + "/subscribe"
	WebPushUnsubscribeURL = WebPushSubscribeURL + "/{%s}"
	WebPushVAPIDKeyURL    = WebPushURL + "/vapid-key"
)

func SetupHandlers(
	apiMux *mux.Router,
	cookiesConfig config.CookiesConfig,
	natsConfig config.NATSConfig,
	usersUseCases interfaces.UsersUseCases,
	authUseCases interfaces.AuthUseCases,
	chatsUseCases interfaces.ChatsUseCases,
	messagesUseCases interfaces.MessagesUseCases,
	settingsUseCases interfaces.SettingsUseCases,
	webPushSubscriptionsUseCases interfaces.WebPushSubscriptionsUseCases,
	logger logging.Logger,
	upgrader interfaces.Upgrader,
	natsPublisher customnats.Publisher,
	vapidPublicKey string,
) {
	getMux := apiMux.Methods(http.MethodGet).Subrouter()
	getMux.Handle(UsersURL, users.Handler(usersUseCases))
	getMux.Handle(MeURL, me.Handler(usersUseCases))
	getMux.Handle(
		fmt.Sprintf(GetUserByIDURL, common.IDRouteKey),
		user_by_id.Handler(usersUseCases),
	)

	websocketHandler := ws.New(
		upgrader,
		usersUseCases,
		chatsUseCases,
		messagesUseCases,
		logger,
		natsPublisher,
		natsConfig,
	)

	getMux.Handle(WebsocketURL, http.HandlerFunc(websocketHandler.Handle))
	getMux.Handle(SettingsURL, get_settings.Handler(settingsUseCases))
	getMux.Handle(ChatsURL, user_chats.Handler(chatsUseCases))
	getMux.Handle(
		fmt.Sprintf(GetChatMessagesURL, common.IDRouteKey),
		chat_messages.Handler(messagesUseCases),
	)
	getMux.Handle(
		fmt.Sprintf(VerifyEmailURL, verify_email.TokenRouteKey),
		verify_email.Handler(authUseCases),
	)
	getMux.Handle(WebPushVAPIDKeyURL, vapid_key.Handler(vapidPublicKey))

	postMux := apiMux.Methods(http.MethodPost).Subrouter()
	postMux.Handle(UsersURL, register.Handler(authUseCases))
	postMux.Handle(SessionsURL, login.Handler(authUseCases, cookiesConfig))
	postMux.Handle(ChangePasswordURL, change_password.Handler(authUseCases))
	postMux.Handle(
		SendVerifyEmailMessageURL,
		send_verify_email_message.Handler(authUseCases),
	)
	postMux.Handle(
		fmt.Sprintf(ForgetPasswordURL, forget_password.TokenRouteKey),
		forget_password.Handler(authUseCases),
	)
	postMux.Handle(
		SendForgetPasswordURL,
		send_forget_password_message.Handler(authUseCases),
	)
	postMux.Handle(ChatsURL, create.Handler(chatsUseCases))
	postMux.Handle(WebPushSubscribeURL, subscribe.Handler(webPushSubscriptionsUseCases))

	putMux := apiMux.Methods(http.MethodPut).Subrouter()
	putMux.Handle(MeURL, update.Handler(usersUseCases))
	putMux.Handle(SettingsURL, update_settings.Handler(settingsUseCases))
	putMux.Handle(SessionsURL, refresh_tokens.Handler(authUseCases, cookiesConfig))

	deleteMux := apiMux.Methods(http.MethodDelete).Subrouter()
	deleteMux.Handle(AllSessionsURL, logout_all.Handler(authUseCases, cookiesConfig))
	deleteMux.Handle(SessionsURL, logout.Handler(authUseCases, cookiesConfig))
	deleteMux.Handle(
		fmt.Sprintf(WebPushUnsubscribeURL, common.IDRouteKey),
		unsubscribe.Handler(webPushSubscriptionsUseCases),
	)
	deleteMux.Handle(
		fmt.Sprintf(DeleteMessageURL, common.IDRouteKey),
		delete_message.Handler(messagesUseCases, websocketHandler),
	)
}
