package http

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/login"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/docs"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/logging"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
	loggingmiddleware "github.com/DKhorkov/libs/middlewares/http/logging"
	metricsmiddleware "github.com/DKhorkov/libs/middlewares/http/metrics"
	requestidmiddleware "github.com/DKhorkov/libs/middlewares/http/request_id"
	tracingmiddleware "github.com/DKhorkov/libs/middlewares/http/tracing"
	customnats "github.com/DKhorkov/libs/nats"
	"github.com/DKhorkov/libs/security"
	"github.com/DKhorkov/libs/tracing"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Controller struct {
	server *http.Server
	logger logging.Logger
	host   string
	port   string
}

func New(
	httpConfig config.HTTPConfig,
	corsConfig config.CORSConfig,
	docsConfig config.DocsConfig,
	cookiesConfig config.CookiesConfig,
	natsConfig config.NATSConfig,
	usersUseCases interfaces.UsersUseCases,
	authUseCases interfaces.AuthUseCases,
	chatsUseCases interfaces.ChatsUseCases,
	messagesUseCases interfaces.MessagesUseCases,
	reactionsUseCases interfaces.ReactionsUseCases,
	settingsUseCases interfaces.SettingsUseCases,
	webPushSubscriptionsUseCases interfaces.WebPushSubscriptionsUseCases,
	fileStorageUseCases interfaces.FileStorageUseCases,
	logger logging.Logger,
	traceProvider tracing.Provider,
	upgrader interfaces.Upgrader,
	natsPublisher customnats.Publisher,
	spanConfig tracing.SpanConfig,
	securityConfig security.Config,
	sensitiveFields []string,
	vapidPublicKey string,
) (*Controller, error) {
	rootMux := mux.NewRouter()
	rootMux.Use(tracingmiddleware.Middleware(traceProvider, spanConfig))
	rootMux.Use(metricsmiddleware.Middleware)
	rootMux.Use(requestidmiddleware.Middleware)
	rootMux.Use(loggingmiddleware.Middleware(logger, sensitiveFields...))
	rootMux.Use(
		authmiddleware.Middleware(
			login.AccessTokenCookieName,
			securityConfig,
			[]authmiddleware.IgnoreURL{
				{
					Path:    regexp.MustCompile(`^` + docs.URL + `$`),
					Methods: []string{http.MethodGet},
				},
				{
					Path: regexp.MustCompile(
						`^` + fmt.Sprintf(handlers.SwaggerURL, docsConfig.Filepath) + `$`,
					),
					Methods: []string{http.MethodGet},
				},
				{
					Path:    regexp.MustCompile(`^` + handlers.APIPrefix + api.SessionsURL + `$`),
					Methods: []string{http.MethodPost, http.MethodPut},
				},
				// Разделяем регистрацию и получение пользователей из-за query параметров на получение
				{
					Path: regexp.MustCompile(
						`^` + handlers.APIPrefix + api.UsersURL + `(?:\?[^ ]*)?$`,
					),
					Methods: []string{http.MethodGet},
				},
				{
					Path:    regexp.MustCompile(`^` + handlers.APIPrefix + api.UsersURL + `$`),
					Methods: []string{http.MethodPost},
				},
				{
					Path: regexp.MustCompile(
						`^` + handlers.APIPrefix + strings.ReplaceAll(
							api.GetUserByIDURL,
							"{%s}",
							"",
						) + `(\d+)$`,
					),
					Methods: []string{http.MethodGet},
				},
				{
					Path: regexp.MustCompile(
						`^` + handlers.APIPrefix + strings.ReplaceAll(
							api.ForgetPasswordURL,
							"{%s}",
							"",
						) + `(\d{6})$`,
					),
					Methods: []string{http.MethodPost},
				},
				{
					Path: regexp.MustCompile(
						`^` + handlers.APIPrefix + api.SendForgetPasswordURL + `$`,
					),
					Methods: []string{http.MethodPost},
				},
				{
					Path: regexp.MustCompile(
						`^` + handlers.APIPrefix + api.SendVerifyEmailMessageURL + `$`,
					),
					Methods: []string{http.MethodPost},
				},
				{
					Path: regexp.MustCompile(
						`^` + handlers.APIPrefix + strings.ReplaceAll(
							api.VerifyEmailURL,
							"{%s}",
							"",
						) + `(\d{6})$`,
					),
					Methods: []string{http.MethodGet},
				},
				{
					Path: regexp.MustCompile(
						`^` + handlers.APIPrefix + api.WebPushVAPIDKeyURL + `$`,
					),
					Methods: []string{http.MethodGet},
				},
				{
					Path:    regexp.MustCompile(`^` + handlers.APIPrefix + api.ReactionsURL + `$`),
					Methods: []string{http.MethodGet},
				},
				{
					Path: regexp.MustCompile(
						`^` + handlers.APIPrefix + strings.ReplaceAll(
							api.FileDownloadURL,
							"{%s}",
							"",
						) + `(.+)$`,
					),
					Methods: []string{http.MethodGet},
				},
				{
					Path:    regexp.MustCompile(`^` + handlers.WebPrefix),
					Methods: []string{http.MethodGet},
				},
			}...,
		),
	)

	handlers.SetupHandlers(
		rootMux,
		docsConfig,
		cookiesConfig,
		natsConfig,
		usersUseCases,
		authUseCases,
		chatsUseCases,
		messagesUseCases,
		reactionsUseCases,
		settingsUseCases,
		webPushSubscriptionsUseCases,
		fileStorageUseCases,
		logger,
		upgrader,
		natsPublisher,
		vapidPublicKey,
	)

	httpHandler := cors.New(
		cors.Options{
			AllowedOrigins:   corsConfig.AllowedOrigins,
			AllowedMethods:   corsConfig.AllowedMethods,
			AllowedHeaders:   corsConfig.AllowedHeaders,
			MaxAge:           corsConfig.MaxAge,
			AllowCredentials: corsConfig.AllowCredentials,
		},
	).Handler(rootMux)

	addr := net.JoinHostPort(httpConfig.Host, httpConfig.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      httpHandler,
		IdleTimeout:  httpConfig.IdleTimeout,
		ReadTimeout:  httpConfig.ReadTimeout,
		WriteTimeout: httpConfig.WriteTimeout,
	}

	return &Controller{
		server: server,
		logger: logger,
		host:   httpConfig.Host,
		port:   httpConfig.Port,
	}, nil
}

func (c *Controller) Run() {
	addr := fmt.Sprintf("%s:%s", c.host, c.port)
	logging.LogInfo(
		c.logger,
		"Ready to serve at "+addr,
	)

	err := c.server.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		logging.LogError(c.logger, "HTTP server error", err)
	}

	logging.LogInfo(c.logger, "Stopped serving new connections.")
}

func (c *Controller) Stop() {
	// Stops accepting new requests and processes already received requests:
	err := c.server.Shutdown(context.Background())
	if err != nil {
		logging.LogError(c.logger, "HTTP shutdown error", err)
	}

	logging.LogInfo(c.logger, "Graceful shutdown completed.")
}
