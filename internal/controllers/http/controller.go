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
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth/login"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/docs"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/logging"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
	loggingmiddleware "github.com/DKhorkov/libs/middlewares/http/logging"
	metricsmiddleware "github.com/DKhorkov/libs/middlewares/http/metrics"
	requestidmiddleware "github.com/DKhorkov/libs/middlewares/http/request_id"
	tracingmiddleware "github.com/DKhorkov/libs/middlewares/http/tracing"
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
	usersUseCases interfaces.UsersUseCases,
	authUseCases interfaces.AuthUseCases,
	chatsUseCases interfaces.ChatsUseCases,
	messagesUseCases interfaces.MessagesUseCases,
	logger logging.Logger,
	traceProvider tracing.Provider,
	upgrader interfaces.Upgrader,
	spanConfig tracing.SpanConfig,
	securityConfig security.Config,
	sensitiveFields []string,
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
					Path:    regexp.MustCompile(`^` + handlers.SessionsURL + `$`),
					Methods: []string{http.MethodPost, http.MethodPut},
				},
				// Разделяем регистрацию и получение пользователей из-за query параметров на получение
				{
					Path: regexp.MustCompile(
						`^` + handlers.UsersURL + `(?:\?[^ ]*)?$`,
					),
					Methods: []string{http.MethodGet},
				},
				{
					Path:    regexp.MustCompile(`^` + handlers.UsersURL + `$`),
					Methods: []string{http.MethodPost},
				},
				{
					Path: regexp.MustCompile(
						`^` + strings.ReplaceAll(handlers.GetUserByIDURL, "{%s}", "") + `(\d+)$`,
					),
					Methods: []string{http.MethodGet},
				},
				{
					Path: regexp.MustCompile(
						`^` + strings.ReplaceAll(handlers.ForgetPasswordURL, "{%s}", "") + `(.+)$`,
					),
					Methods: []string{http.MethodPost},
				},
				{
					Path:    regexp.MustCompile(`^` + handlers.SendForgetPasswordURL + `$`),
					Methods: []string{http.MethodPost},
				},
				{
					Path:    regexp.MustCompile(`^` + handlers.SendVerifyEmailMessageURL + `$`),
					Methods: []string{http.MethodPost},
				},
				{
					Path: regexp.MustCompile(
						`^` + strings.ReplaceAll(handlers.VerifyEmailURL, "{%s}", "") + `(.+)$`,
					),
					Methods: []string{http.MethodGet},
				},
			}...,
		),
	)

	handlers.SetupHandlers(
		rootMux,
		docsConfig,
		cookiesConfig,
		usersUseCases,
		authUseCases,
		chatsUseCases,
		messagesUseCases,
		logger,
		upgrader,
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
