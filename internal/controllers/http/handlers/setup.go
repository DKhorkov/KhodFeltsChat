package handlers

import (
	"fmt"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	middlewares "github.com/DKhorkov/libs/middlewares/http"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/users"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

const (
	docsURL = "/docs"

	sessionsURL = "/sessions"

	usersURL                  = "/users"
	meURL                     = usersURL + "/me"
	getUserByIDURL            = usersURL + "/{%s}"
	passwordURL               = usersURL + "/password"
	changePasswordURL         = passwordURL + "/change"
	sendForgetPasswordURL     = passwordURL + "/forget"
	forgetPasswordURL         = sendForgetPasswordURL + "/{%s}"
	sendVerifyEmailMessageURL = usersURL + "/email/verify"
	verifyEmailURL            = sendVerifyEmailMessageURL + "/{%s}"
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
	getMux.Handle(usersURL, users.GetUsersHandler(usersUseCases))
	getMux.Handle(meURL, users.GetMeHandler(usersUseCases))
	getMux.Handle(fmt.Sprintf(getUserByIDURL, users.IDRouteKey), users.GetUserByIDHandler(usersUseCases))

	swaggerURL := "/" + docsConfig.Filepath
	opts := middleware.RedocOpts{SpecURL: swaggerURL}                    // Устанавливаем название юрла файла для обслуживания сваггера
	sh := middleware.Redoc(opts, nil)                                    // Мидлварь для обаботки файла при переходе по юрлу документации
	getMux.Handle(docsURL, sh)                                           // Устанавливаем юрл для получения документации
	getMux.Handle(swaggerURL, http.FileServer(http.Dir(docsConfig.Dir))) // Связываем установленный юрл с отдачей файла

	postMux := rootMux.Methods(http.MethodPost).Subrouter()
	postMux.Handle(usersURL, auth.RegisterHandler(authUseCases))
	postMux.Handle(sessionsURL, auth.LoginHandler(authUseCases, cookiesConfig))
	postMux.Handle(changePasswordURL, auth.ChangePasswordHandler(authUseCases))
	postMux.Handle(fmt.Sprintf(verifyEmailURL, auth.VerifyEmailTokenRouteKey), auth.VerifyEmailHandler(authUseCases))
	postMux.Handle(sendVerifyEmailMessageURL, auth.SendVerifyEmailMessageHandler(authUseCases))
	postMux.Handle(fmt.Sprintf(forgetPasswordURL, auth.ForgetPasswordTokenRouteKey), auth.ForgetPasswordHandler(authUseCases))
	postMux.Handle(sendForgetPasswordURL, auth.SendForgetPasswordMessageHandler(authUseCases))

	putMux := rootMux.Methods(http.MethodPut).Subrouter()
	putMux.Handle(meURL, users.UpdateCurrentUserHandler(usersUseCases))
	putMux.Handle(sessionsURL, auth.RefreshTokensHandler(authUseCases, cookiesConfig))

	deleteMux := rootMux.Methods(http.MethodDelete).Subrouter()
	deleteMux.Handle(sessionsURL, auth.LogoutHandler(authUseCases))
}
