package handlers

import (
	"fmt"
	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/interfaces"
	middlewares "github.com/DKhorkov/libs/middlewares/http"
	"github.com/go-openapi/runtime/middleware"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

const (
	docsURL = "/docs"
)

func SetupHandlers(
	rootMux *mux.Router,
	docsConfig config.DocsConfig,
	usersUseCases interfaces.UsersUseCases,
	authUseCases interfaces.AuthUseCases,
) {
	rootMux.NotFoundHandler = http.HandlerFunc(DefaultHandler)
	rootMux.MethodNotAllowedHandler = http.HandlerFunc(NotAllowedHandler)

	getMux := rootMux.Methods(http.MethodGet).Subrouter()
	getMux.Handle(middlewares.MetricsURLPath, promhttp.Handler())
	//getMux.Handle("/entries", handlers.ListHandler(useCases))
	//getMux.Handle(fmt.Sprintf("/entries/{%s}", handlers.SearchKey), handlers.SearchHandler(useCases))

	swaggerURL := fmt.Sprintf("/%s", docsConfig.Filepath)
	opts := middleware.RedocOpts{SpecURL: swaggerURL}                    // Устанавливаем название юрла файла для обслуживания сваггера
	sh := middleware.Redoc(opts, nil)                                    // Мидлварь для обаботки файла при переходе по юрлу документации
	getMux.Handle(docsURL, sh)                                           // Устанавливаем юрл для получения документации
	getMux.Handle(swaggerURL, http.FileServer(http.Dir(docsConfig.Dir))) // Связываем установленный юрл с отдачей файла

	//postMux := rootMux.Methods(http.MethodPost).Subrouter()
	//postMux.Handle("/entries", handlers.InsertHandler(useCases))

	//deleteMux := rootMux.Methods(http.MethodDelete).Subrouter()
	//deleteMux.Handle(fmt.Sprintf("/entries/{%s}", handlers.DeleteKey), handlers.DeleteHandler(useCases))
}
