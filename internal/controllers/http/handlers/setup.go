package handlers

import (
	"fmt"
	"net/http"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/users"
	"github.com/DKhorkov/kfc/internal/interfaces"
	middlewares "github.com/DKhorkov/libs/middlewares/http"
	"github.com/go-openapi/runtime/middleware"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	DocsURL = "/docs"

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
)

func SetupHandlers(
	rootMux *mux.Router,
	docsConfig config.DocsConfig,
	cookiesConfig config.CookiesConfig,
	usersUseCases interfaces.UsersUseCases,
	authUseCases interfaces.AuthUseCases,
) {
	rootMux.NotFoundHandler = http.HandlerFunc(DefaultHandler)
	rootMux.MethodNotAllowedHandler = http.HandlerFunc(NotAllowedHandler)

	getMux := rootMux.Methods(http.MethodGet).Subrouter()
	getMux.Handle(middlewares.MetricsURLPath, promhttp.Handler())
	getMux.Handle(UsersURL, users.GetUsersHandler(usersUseCases))
	getMux.Handle(MeURL, users.GetMeHandler(usersUseCases))
	getMux.Handle(
		fmt.Sprintf(GetUserByIDURL, users.IDRouteKey),
		users.GetUserByIDHandler(usersUseCases),
	)

	swaggerURL := "/" + docsConfig.Filepath
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

	putMux := rootMux.Methods(http.MethodPut).Subrouter()
	putMux.Handle(MeURL, users.UpdateCurrentUserHandler(usersUseCases))
	putMux.Handle(SessionsURL, auth.RefreshTokensHandler(authUseCases, cookiesConfig))

	deleteMux := rootMux.Methods(http.MethodDelete).Subrouter()
	deleteMux.Handle(SessionsURL, auth.LogoutHandler(authUseCases))
}
