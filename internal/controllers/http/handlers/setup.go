package handlers

import (
	"fmt"
	"net/http"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/default"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/docs"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/not_allowed"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/web"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/logging"
	metricsmiddleware "github.com/DKhorkov/libs/middlewares/http/metrics"
	customnats "github.com/DKhorkov/libs/nats"
	"github.com/go-openapi/runtime/middleware"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	APIPrefix  = "/api"
	WebPrefix  = "/web"
	SwaggerURL = "/%s"
)

func SetupHandlers(
	rootMux *mux.Router,
	docsConfig config.DocsConfig,
	cookiesConfig config.CookiesConfig,
	natsConfig config.NATSConfig,
	usersUseCases interfaces.UsersUseCases,
	authUseCases interfaces.AuthUseCases,
	chatsUseCases interfaces.ChatsUseCases,
	messagesUseCases interfaces.MessagesUseCases,
	settingsUseCases interfaces.SettingsUseCases,
	pushSubscriptionsUseCases interfaces.PushSubscriptionsUseCases,
	logger logging.Logger,
	upgrader interfaces.Upgrader,
	natsPublisher customnats.Publisher,
	vapidPublicKey string,
) {
	rootMux.NotFoundHandler = http.HandlerFunc(default_handler.Handler)
	rootMux.MethodNotAllowedHandler = http.HandlerFunc(not_allowed.Handler)

	// Metrics:
	rootMux.Methods(http.MethodGet).Subrouter().Handle(
		metricsmiddleware.MetricsURLPath,
		promhttp.Handler(),
	)

	// Docs (Swagger):
	swaggerURL := fmt.Sprintf(SwaggerURL, docsConfig.Filepath)
	opts := middleware.RedocOpts{
		SpecURL: swaggerURL,
	}
	sh := middleware.Redoc(opts, nil)
	getMux := rootMux.Methods(http.MethodGet).Subrouter()
	getMux.Handle(docs.URL, sh)
	getMux.Handle(swaggerURL, http.FileServer(http.Dir(docsConfig.Dir)))

	// API subrouter:
	apiMux := rootMux.PathPrefix(APIPrefix).Subrouter()
	api.SetupHandlers(
		apiMux,
		cookiesConfig,
		natsConfig,
		usersUseCases,
		authUseCases,
		chatsUseCases,
		messagesUseCases,
		settingsUseCases,
		pushSubscriptionsUseCases,
		logger,
		upgrader,
		natsPublisher,
		vapidPublicKey,
	)

	// WEB subrouter:
	webMux := rootMux.PathPrefix(WebPrefix).Subrouter()
	web.SetupHandlers(webMux)
}
