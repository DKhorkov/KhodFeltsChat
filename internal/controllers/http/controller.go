package http

import (
	"context"
	"errors"
	"fmt"
	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/logging"
	middlewares "github.com/DKhorkov/libs/middlewares/http"
	"github.com/DKhorkov/libs/tracing"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"net/http"
)

type Controller struct {
	server *http.Server
	logger logging.Logger
	host   string
	port   int
}

func New(
	httpConfig config.HTTPConfig,
	corsConfig config.CORSConfig,
	docsConfig config.DocsConfig,
	usersUseCases interfaces.UsersUseCases,
	authUseCases interfaces.AuthUseCases,
	logger logging.Logger,
	traceProvider tracing.Provider,
	spanConfig tracing.SpanConfig,
) (*Controller, error) {
	rootMux := mux.NewRouter()
	rootMux.Use(middlewares.TracingMiddleware(traceProvider, spanConfig))
	rootMux.Use(middlewares.MetricsMiddleware)

	handlers.SetupHandlers(
		rootMux,
		docsConfig,
		usersUseCases,
		authUseCases,
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

	addr := fmt.Sprintf("%s:%d", httpConfig.Host, httpConfig.Port)
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
	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	logging.LogInfo(
		c.logger,
		fmt.Sprintf("Ready to serve at %s", addr),
	)

	if err := c.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
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
