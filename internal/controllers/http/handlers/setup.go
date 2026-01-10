package handlers

import (
	"fmt"
	"net/http"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/chats"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/messages"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/users"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/ws"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/logging"
	middlewares "github.com/DKhorkov/libs/middlewares/http"
	"github.com/go-openapi/runtime/middleware"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	DocsURL    = "/docs"
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
	rootMux.NotFoundHandler = http.HandlerFunc(DefaultHandler)
	rootMux.MethodNotAllowedHandler = http.HandlerFunc(NotAllowedHandler)

	getMux := rootMux.Methods(http.MethodGet).Subrouter()
	getMux.Handle(middlewares.MetricsURLPath, promhttp.Handler())
	getMux.Handle(UsersURL, users.GetUsersHandler(usersUseCases))
	getMux.Handle(MeURL, users.GetMeHandler(usersUseCases))
	getMux.Handle(
		fmt.Sprintf(GetUserByIDURL, common.IDRouteKey),
		users.GetUserByIDHandler(usersUseCases),
	)

	websocketHandler := ws.New(
		upgrader,
		usersUseCases,
		chatsUseCases,
		messagesUseCases,
		logger,
	)

	getMux.Handle(WebsocketURL, http.HandlerFunc(websocketHandler.Handle))
	getMux.Handle(ChatsURL, chats.GetUserChatsHandler(chatsUseCases))
	getMux.Handle(
		fmt.Sprintf(GetChatMessagesURL, common.IDRouteKey),
		messages.GetChatMessagesHandler(messagesUseCases),
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
		DocsURL,
		sh,
	) // Устанавливаем юрл для получения документации
	getMux.Handle(
		swaggerURL,
		http.FileServer(http.Dir(docsConfig.Dir)),
	) // Связываем установленный юрл с отдачей файла

	postMux := rootMux.Methods(http.MethodPost).Subrouter()
	postMux.Handle(UsersURL, auth.RegisterHandler(authUseCases))
	postMux.Handle(SessionsURL, auth.LoginHandler(authUseCases, cookiesConfig))
	postMux.Handle(ChangePasswordURL, auth.ChangePasswordHandler(authUseCases))
	postMux.Handle(
		fmt.Sprintf(VerifyEmailURL, auth.VerifyEmailTokenRouteKey),
		auth.VerifyEmailHandler(authUseCases),
	)
	postMux.Handle(SendVerifyEmailMessageURL, auth.SendVerifyEmailMessageHandler(authUseCases))
	postMux.Handle(
		fmt.Sprintf(ForgetPasswordURL, auth.ForgetPasswordTokenRouteKey),
		auth.ForgetPasswordHandler(authUseCases),
	)
	postMux.Handle(SendForgetPasswordURL, auth.SendForgetPasswordMessageHandler(authUseCases))
	postMux.Handle(ChatsURL, chats.CreateChatHandler(chatsUseCases))

	putMux := rootMux.Methods(http.MethodPut).Subrouter()
	putMux.Handle(MeURL, users.UpdateCurrentUserHandler(usersUseCases))
	putMux.Handle(SessionsURL, auth.RefreshTokensHandler(authUseCases, cookiesConfig))

	deleteMux := rootMux.Methods(http.MethodDelete).Subrouter()
	deleteMux.Handle(SessionsURL, auth.LogoutHandler(authUseCases))
}
