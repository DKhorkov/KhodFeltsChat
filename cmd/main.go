package main

import (
	"context"
	"database/sql"
	"github.com/DKhorkov/kfc/internal/app"
	"github.com/DKhorkov/kfc/internal/config"
	controllers "github.com/DKhorkov/kfc/internal/controllers/http"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/repositories"
	"github.com/DKhorkov/kfc/internal/services"
	"github.com/DKhorkov/kfc/internal/uow"
	"github.com/DKhorkov/kfc/internal/usecases"
	"github.com/DKhorkov/libs/db/postgresql"
	"github.com/DKhorkov/libs/loadenv"
	"github.com/DKhorkov/libs/logging"
	"github.com/DKhorkov/libs/tracing"
)

func main() {
	// Инициализируем переменные окружения для дальнейшего считывания в конфиге:
	loadenv.Init()

	cfg := config.New()
	logger := logging.New(
		cfg.Logging.Level,
		cfg.Logging.LogFilePath,
	)

	pg, err := postgresql.New(
		postgresql.BuildDsn(cfg.Database),
		cfg.Database.Driver,
		logger,
		postgresql.WithMaxOpenConnections(cfg.Database.Pool.MaxOpenConnections),
		postgresql.WithMaxIdleConnections(cfg.Database.Pool.MaxIdleConnections),
		postgresql.WithMaxConnectionLifetime(cfg.Database.Pool.MaxConnectionLifetime),
		postgresql.WithMaxConnectionIdleTime(cfg.Database.Pool.MaxConnectionIdleTime),
	)
	if err != nil {
		panic(err)
	}

	traceProvider, err := tracing.New(cfg.Tracing.Server)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err = traceProvider.Shutdown(context.Background()); err != nil {
			logging.LogError(logger, "Error shutting down tracer", err)
		}
	}()

	unitOfWork := uow.New(pg)

	usersService := services.NewUsersService(
		unitOfWork,
		func(tx *sql.Tx) interfaces.UsersRepository {
			return repositories.NewUsersRepository(tx)
		},
	)

	authService := services.NewAuthService(
		unitOfWork,
		func(tx *sql.Tx) interfaces.AuthRepository {
			return repositories.NewAuthRepository(tx)
		},
		func(tx *sql.Tx) interfaces.UsersRepository {
			return repositories.NewUsersRepository(tx)
		},
		func() interfaces.EmailsRepository {
			return repositories.NewEmailsRepository(cfg.Email.SMTP)
		},
	)

	usersUseCases := usecases.NewUsersUseCases(usersService, cfg.Security, cfg.Validation)
	authUseCases := usecases.NewAuthUseCases(authService, usersService, cfg.Security, cfg.Validation)

	c, err := controllers.New(
		cfg.HTTP,
		cfg.CORS,
		cfg.Docs,
		cfg.Cookies,
		usersUseCases,
		authUseCases,
		logger,
		traceProvider,
		cfg.Tracing.Spans.Root,
	)

	if err != nil {
		panic(err)
	}

	application := app.New(c)
	application.Run()
}
