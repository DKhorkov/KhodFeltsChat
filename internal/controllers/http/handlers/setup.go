package handlers

import (
	"fmt"
	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/users"
	"github.com/DKhorkov/kfc/internal/interfaces"
	middlewares "github.com/DKhorkov/libs/middlewares/http"
	"github.com/go-openapi/runtime/middleware"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

const (
	docsURL        = "/docs"
	authURL        = "/auth"
	registerURL    = authURL + "/register"
	loginURL       = authURL + "/login"
	usersURL       = "/users"
	getMeURL       = usersURL + "/me"
	getUserByIDURL = usersURL + "/{%s}"
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
	getMux.Handle(getMeURL, users.GetMeHandler(usersUseCases))
	getMux.Handle(fmt.Sprintf(getUserByIDURL, users.IDRouteKey), users.GetUserByIDHandler(usersUseCases))

	swaggerURL := fmt.Sprintf("/%s", docsConfig.Filepath)
	opts := middleware.RedocOpts{SpecURL: swaggerURL}                    // Устанавливаем название юрла файла для обслуживания сваггера
	sh := middleware.Redoc(opts, nil)                                    // Мидлварь для обаботки файла при переходе по юрлу документации
	getMux.Handle(docsURL, sh)                                           // Устанавливаем юрл для получения документации
	getMux.Handle(swaggerURL, http.FileServer(http.Dir(docsConfig.Dir))) // Связываем установленный юрл с отдачей файла

	postMux := rootMux.Methods(http.MethodPost).Subrouter()
	postMux.Handle(registerURL, auth.RegisterHandler(authUseCases))
	postMux.Handle(loginURL, auth.LoginHandler(authUseCases, cookiesConfig))

	//deleteMux := rootMux.Methods(http.MethodDelete).Subrouter()
	//deleteMux.Handle(fmt.Sprintf("/entries/{%s}", handlers.DeleteKey), handlers.DeleteHandler(useCases))
}
