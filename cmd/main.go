package main

import (
	"context"

	"github.com/DKhorkov/kfc/internal/app"
	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/contentbuilders"
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
	"github.com/gorilla/websocket"
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
		err = traceProvider.Shutdown(context.Background())
		if err != nil {
			logging.LogError(logger, "Error shutting down tracer", err)
		}
	}()

	contentBuilders := interfaces.ContentBuilders{
		VerifyEmail: contentbuilders.NewVerifyEmailContentBuilder(
			cfg.Email.VerifyEmailURL,
		),
		ForgetPassword: contentbuilders.NewForgetPasswordContentBuilder(
			cfg.Email.ForgetPasswordURL,
		),
	}

	unitOfWork := uow.New(pg)

	usersService := services.NewUsersService(
		unitOfWork,
		func(tx postgresql.Transaction) interfaces.UsersRepository {
			return repositories.NewUsersRepository(tx, logger)
		},
	)

	authService := services.NewAuthService(
		unitOfWork,
		func(tx postgresql.Transaction) interfaces.AuthRepository {
			return repositories.NewAuthRepository(tx)
		},
		func(tx postgresql.Transaction) interfaces.UsersRepository {
			return repositories.NewUsersRepository(tx, logger)
		},
		func() interfaces.EmailsRepository {
			return repositories.NewEmailsRepository(cfg.Email.SMTP, contentBuilders)
		},
	)

	chatsService := services.NewChatsService(
		unitOfWork,
		func(tx postgresql.Transaction) interfaces.ChatsRepository {
			return repositories.NewChatsRepository(tx, logger)
		},
		func(tx postgresql.Transaction) interfaces.MessagesRepository {
			return repositories.NewMessagesRepository(tx, logger)
		},
	)

	messagesService := services.NewMessagesService(
		unitOfWork,
		func(tx postgresql.Transaction) interfaces.ChatsRepository {
			return repositories.NewChatsRepository(tx, logger)
		},
		func(tx postgresql.Transaction) interfaces.MessagesRepository {
			return repositories.NewMessagesRepository(tx, logger)
		},
	)

	usersUseCases := usecases.NewUsersUseCases(usersService, cfg.Security, cfg.Validation)
	messagesUseCases := usecases.NewMessagesUseCases(messagesService, chatsService, usersService)
	chatsUseCases := usecases.NewChatsUseCases(chatsService, usersService)
	authUseCases := usecases.NewAuthUseCases(
		authService,
		usersService,
		cfg.Security,
		cfg.Validation,
	)

	upgrader := &websocket.Upgrader{
		HandshakeTimeout: cfg.Websocket.HandshakeTimeout,
	}

	c, err := controllers.New(
		cfg.HTTP,
		cfg.CORS,
		cfg.Docs,
		cfg.Cookies,
		usersUseCases,
		authUseCases,
		chatsUseCases,
		messagesUseCases,
		logger,
		traceProvider,
		upgrader,
		cfg.Tracing.Spans.Root,
		cfg.Security,
		[]string{ // Чувствительная информация, которая не должна быть заллогирована
			"email",
			"password",
			"oldPassword",
			"newPassword",
		},
	)
	if err != nil {
		panic(err)
	}

	application := app.New(c)
	application.Run()
}
