package main

import (
	"context"

	"github.com/DKhorkov/kfc/internal/app"
	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/contentbuilders/forget_password"
	"github.com/DKhorkov/kfc/internal/contentbuilders/verify_email"
	controllers "github.com/DKhorkov/kfc/internal/controllers/http"
	"github.com/DKhorkov/kfc/internal/interfaces"
	authrepository "github.com/DKhorkov/kfc/internal/repositories/auth"
	chatsrepository "github.com/DKhorkov/kfc/internal/repositories/chats"
	emailsrepository "github.com/DKhorkov/kfc/internal/repositories/emails"
	messagesrepository "github.com/DKhorkov/kfc/internal/repositories/messages"
	usersrepository "github.com/DKhorkov/kfc/internal/repositories/users"
	authservice "github.com/DKhorkov/kfc/internal/services/auth"
	chatsservice "github.com/DKhorkov/kfc/internal/services/chats"
	messagesservice "github.com/DKhorkov/kfc/internal/services/messages"
	usersservice "github.com/DKhorkov/kfc/internal/services/users"
	"github.com/DKhorkov/kfc/internal/uow"
	authusecases "github.com/DKhorkov/kfc/internal/usecases/auth"
	chatsusecases "github.com/DKhorkov/kfc/internal/usecases/chats"
	messagesusecases "github.com/DKhorkov/kfc/internal/usecases/messages"
	usersusecases "github.com/DKhorkov/kfc/internal/usecases/users"
	"github.com/DKhorkov/libs/cache"
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

	cacheProvider, err := cache.New(
		cache.WithHost(cfg.Cache.Host),
		cache.WithPort(cfg.Cache.Port),
		cache.WithPassword(cfg.Cache.Password),
	)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err = cacheProvider.Close(); err != nil {
			logging.LogError(logger, "Error shutting down cache", err)
		}
	}()

	contentBuilders := interfaces.ContentBuilders{
		VerifyEmail: verify_email.New(
			cfg.Email.VerifyEmailURL,
		),
		ForgetPassword: forget_password.New(),
	}

	unitOfWork := uow.NewTraceDecorator(
		traceProvider,
		cfg.Tracing.Spans.UnitOfWork,
		uow.New(pg),
	)

	usersService := usersservice.NewTraceDecorator(
		traceProvider,
		cfg.Tracing.Spans.Services.Users,
		usersservice.New(
			unitOfWork,
			func(tx postgresql.Transaction) interfaces.UsersRepository {
				return usersrepository.NewTraceDecorator(
					traceProvider,
					cfg.Tracing.Spans.Repositories.Users,
					usersrepository.New(tx, logger),
				)
			},
		),
	)

	authService := authservice.NewTraceDecorator(
		traceProvider,
		cfg.Tracing.Spans.Services.Auth,
		authservice.New(
			unitOfWork,
			func(tx postgresql.Transaction) interfaces.AuthRepository {
				return authrepository.NewTraceDecorator(
					traceProvider,
					cfg.Tracing.Spans.Repositories.Auth,
					authrepository.New(tx),
				)
			},
			func(tx postgresql.Transaction) interfaces.UsersRepository {
				return usersrepository.NewTraceDecorator(
					traceProvider,
					cfg.Tracing.Spans.Repositories.Users,
					usersrepository.New(tx, logger),
				)
			},
			func() interfaces.EmailsRepository {
				return emailsrepository.NewTraceDecorator(
					traceProvider,
					cfg.Tracing.Spans.Repositories.Emails,
					emailsrepository.New(cfg.Email.SMTP, contentBuilders),
				)
			},
		),
	)

	chatsService := chatsservice.NewTraceDecorator(
		traceProvider,
		cfg.Tracing.Spans.Services.Chats,
		chatsservice.New(
			unitOfWork,
			func(tx postgresql.Transaction) interfaces.ChatsRepository {
				return chatsrepository.NewTraceDecorator(
					traceProvider,
					cfg.Tracing.Spans.Repositories.Chats,
					chatsrepository.New(tx, logger),
				)
			},
			func(tx postgresql.Transaction) interfaces.MessagesRepository {
				return messagesrepository.NewTraceDecorator(
					traceProvider,
					cfg.Tracing.Spans.Repositories.Messages,
					messagesrepository.New(tx, logger),
				)
			},
		),
	)

	messagesService := messagesservice.NewTraceDecorator(
		traceProvider,
		cfg.Tracing.Spans.Services.Messages,
		messagesservice.New(
			unitOfWork,
			func(tx postgresql.Transaction) interfaces.ChatsRepository {
				return chatsrepository.NewTraceDecorator(
					traceProvider,
					cfg.Tracing.Spans.Repositories.Chats,
					chatsrepository.New(tx, logger),
				)
			},
			func(tx postgresql.Transaction) interfaces.MessagesRepository {
				return messagesrepository.NewTraceDecorator(
					traceProvider,
					cfg.Tracing.Spans.Repositories.Messages,
					messagesrepository.New(tx, logger),
				)
			},
		),
	)

	usersUseCases := usersusecases.NewTraceDecorator(
		traceProvider,
		cfg.Tracing.Spans.UseCases.Users,
		usersusecases.New(usersService, cfg.Security, cfg.Validation),
	)

	messagesUseCases := messagesusecases.NewTraceDecorator(
		traceProvider,
		cfg.Tracing.Spans.UseCases.Messages,
		messagesusecases.New(messagesService, chatsService, usersService),
	)

	chatsUseCases := chatsusecases.NewTraceDecorator(
		traceProvider,
		cfg.Tracing.Spans.UseCases.Chats,
		chatsusecases.New(chatsService, usersService),
	)

	authUseCases := authusecases.NewTraceDecorator(
		traceProvider,
		cfg.Tracing.Spans.UseCases.Auth,
		authusecases.NewCacheDecorator(
			cacheProvider,
			logger,
			authusecases.New(
				authService,
				usersService,
				cfg.Security,
				cfg.Validation,
			),
		),
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
