package handlers

import (
	"fmt"
	"net/http"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth/change_password"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth/forget_password"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth/login"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth/logout"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth/refresh_tokens"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth/register"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth/send_forget_password_message"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth/send_verify_email_message"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth/verify_email"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/chats/create"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/chats/user_chats"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	default_handler "github.com/DKhorkov/kfc/internal/controllers/http/handlers/default"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/docs"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/messages/chat_messages"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/not_allowed"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/users/me"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/users/update"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/users/user_by_id"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/users/users"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/ws"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/logging"
	middlewares "github.com/DKhorkov/libs/middlewares/http"
	"github.com/go-openapi/runtime/middleware"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	SwaggerURL = "/%s"

	SessionsURL = "/sessions"

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

	ChatsURL           = "/chats"
	GetChatMessagesURL = ChatsURL + "/{%s}/messages"
)

func SetupHandlers(
	rootMux *mux.Router,
	docsConfig config.DocsConfig,
	cookiesConfig config.CookiesConfig,
	usersUseCases interfaces.UsersUseCases,
	authUseCases interfaces.AuthUseCases,
	chatsUseCases interfaces.ChatsUseCases,
	messagesUseCases interfaces.MessagesUseCases,
	logger logging.Logger,
	upgrader interfaces.Upgrader,
) {
	rootMux.NotFoundHandler = http.HandlerFunc(default_handler.Handler)
	rootMux.MethodNotAllowedHandler = http.HandlerFunc(not_allowed.Handler)

	getMux := rootMux.Methods(http.MethodGet).Subrouter()
	getMux.Handle(middlewares.MetricsURLPath, promhttp.Handler())
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
	)

	getMux.Handle(WebsocketURL, http.HandlerFunc(websocketHandler.Handle))
	getMux.Handle(ChatsURL, user_chats.Handler(chatsUseCases))
	getMux.Handle(
		fmt.Sprintf(GetChatMessagesURL, common.IDRouteKey),
		chat_messages.Handler(messagesUseCases),
	)
	getMux.Handle(
		fmt.Sprintf(VerifyEmailURL, verify_email.TokenRouteKey),
		verify_email.Handler(authUseCases),
	)

	swaggerURL := fmt.Sprintf(SwaggerURL, docsConfig.Filepath)
	opts := middleware.RedocOpts{
		SpecURL: swaggerURL,
	} // Устанавливаем название юрла файла для обслуживания сваггера
	sh := middleware.Redoc(
		opts,
		nil,
	) // Мидлварь для обаботки файла при переходе по юрлу документации
	getMux.Handle(
		docs.URL,
		sh,
	) // Устанавливаем юрл для получения документации
	getMux.Handle(
		swaggerURL,
		http.FileServer(http.Dir(docsConfig.Dir)),
	) // Связываем установленный юрл с отдачей файла

	postMux := rootMux.Methods(http.MethodPost).Subrouter()
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

	putMux := rootMux.Methods(http.MethodPut).Subrouter()
	putMux.Handle(MeURL, update.Handler(usersUseCases))
	putMux.Handle(SessionsURL, refresh_tokens.Handler(authUseCases, cookiesConfig))

	deleteMux := rootMux.Methods(http.MethodDelete).Subrouter()
	deleteMux.Handle(SessionsURL, logout.Handler(authUseCases))
}
