package main

import (
	"context"
	"fmt"

	"github.com/DKhorkov/kfc/internal/app"
	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/contentbuilders/forget_password"
	newmessage "github.com/DKhorkov/kfc/internal/contentbuilders/new_message"
	"github.com/DKhorkov/kfc/internal/contentbuilders/verify_email"
	controllers "github.com/DKhorkov/kfc/internal/controllers/http"
	"github.com/DKhorkov/kfc/internal/interfaces"
	authrepository "github.com/DKhorkov/kfc/internal/repositories/auth"
	chatsrepository "github.com/DKhorkov/kfc/internal/repositories/chats"
	emailsrepository "github.com/DKhorkov/kfc/internal/repositories/emails"
	filestoragerepository "github.com/DKhorkov/kfc/internal/repositories/file_storage"
	messagesrepository "github.com/DKhorkov/kfc/internal/repositories/messages"
	settingsrepository "github.com/DKhorkov/kfc/internal/repositories/settings"
	usersrepository "github.com/DKhorkov/kfc/internal/repositories/users"
	webpushrepository "github.com/DKhorkov/kfc/internal/repositories/web_push"
	webpushsubscriptionsrepository "github.com/DKhorkov/kfc/internal/repositories/web_push_subscriptions"
	authservice "github.com/DKhorkov/kfc/internal/services/auth"
	chatsservice "github.com/DKhorkov/kfc/internal/services/chats"
	filestorageservice "github.com/DKhorkov/kfc/internal/services/file_storage"
	messagesservice "github.com/DKhorkov/kfc/internal/services/messages"
	notificationsservice "github.com/DKhorkov/kfc/internal/services/notifications"
	settingsservice "github.com/DKhorkov/kfc/internal/services/settings"
	usersservice "github.com/DKhorkov/kfc/internal/services/users"
	webpushsubscriptionsservice "github.com/DKhorkov/kfc/internal/services/web_push_subscriptions"
	"github.com/DKhorkov/kfc/internal/uow"
	authusecases "github.com/DKhorkov/kfc/internal/usecases/auth"
	chatsusecases "github.com/DKhorkov/kfc/internal/usecases/chats"
	filestorageusecases "github.com/DKhorkov/kfc/internal/usecases/file_storage"
	messagesusecases "github.com/DKhorkov/kfc/internal/usecases/messages"
	notificaionsusecases "github.com/DKhorkov/kfc/internal/usecases/notifications"
	settingsusecases "github.com/DKhorkov/kfc/internal/usecases/settings"
	usersusecases "github.com/DKhorkov/kfc/internal/usecases/users"
	webpushsubscriptionsusecases "github.com/DKhorkov/kfc/internal/usecases/web_push_subscriptions"
	emailnotificationmessagehandlerbuilder "github.com/DKhorkov/kfc/internal/workers/handlers/builders/email_notification"
	messagehandlerbuildertracingdecorator "github.com/DKhorkov/kfc/internal/workers/handlers/builders/tracing_decorator"
	webpushnotificationmessagehandlerbuilder "github.com/DKhorkov/kfc/internal/workers/handlers/builders/web_push_notification"
	"github.com/DKhorkov/libs/cache"
	"github.com/DKhorkov/libs/db/postgresql"
	"github.com/DKhorkov/libs/loadenv"
	"github.com/DKhorkov/libs/logging"
	customnats "github.com/DKhorkov/libs/nats"
	"github.com/DKhorkov/libs/tracing"
	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
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

	natsPublisher, err := customnats.NewPublisher(
		cfg.NATS.ClientURL,
		nats.Name(cfg.NATS.Publisher.Name),
	)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err = natsPublisher.Close(); err != nil {
			logging.LogError(logger, "Failed to close nats publisher", err)
		}
	}()

	unitOfWork := uow.NewTraceDecorator(
		traceProvider,
		cfg.Tracing.Spans.UnitOfWork,
		uow.New(pg),
	)

	contentBuilders := interfaces.ContentBuilders{
		VerifyEmail: verify_email.New(
			cfg.Email.VerifyEmailURL,
			cacheProvider,
		),
		ForgetPassword: forget_password.New(
			cacheProvider,
		),
		NewMessage: newmessage.New(),
	}

	emailsRepository := emailsrepository.NewTraceDecorator(
		traceProvider,
		cfg.Tracing.Spans.Repositories.Emails,
		emailsrepository.New(cfg.Email.SMTP, contentBuilders),
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

	settingsService := settingsservice.NewTraceDecorator(
		traceProvider,
		cfg.Tracing.Spans.Services.Settings,
		settingsservice.New(
			unitOfWork,
			func(tx postgresql.Transaction) interfaces.SettingsRepository {
				return settingsrepository.NewTraceDecorator(
					traceProvider,
					cfg.Tracing.Spans.Repositories.Settings,
					settingsrepository.New(tx),
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
			func(tx postgresql.Transaction) interfaces.SettingsRepository {
				return settingsrepository.NewTraceDecorator(
					traceProvider,
					cfg.Tracing.Spans.Repositories.Settings,
					settingsrepository.New(tx),
				)
			},
			natsPublisher,
			cfg.NATS,
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

	webPushSubscriptionsService := webpushsubscriptionsservice.NewTraceDecorator(
		traceProvider,
		cfg.Tracing.Spans.Services.WebPushSubscriptions,
		webpushsubscriptionsservice.New(
			unitOfWork,
			func(tx postgresql.Transaction) interfaces.WebPushSubscriptionsRepository {
				return webpushsubscriptionsrepository.NewTraceDecorator(
					traceProvider,
					cfg.Tracing.Spans.Repositories.WebPushSubscriptions,
					webpushsubscriptionsrepository.New(tx),
				)
			},
		),
	)

	webPushRepository := webpushrepository.NewTraceDecorator(
		traceProvider,
		cfg.Tracing.Spans.Repositories.WebPush,
		webpushrepository.New(cfg.WebPush, logger),
	)

	notificationsService := notificationsservice.NewTraceDecorator(
		traceProvider,
		cfg.Tracing.Spans.Services.Notifications,
		notificationsservice.New(
			emailsRepository,
			webPushRepository,
		),
	)

	settingsUseCases := settingsusecases.NewTraceDecorator(
		traceProvider,
		cfg.Tracing.Spans.UseCases.Settings,
		settingsusecases.New(settingsService),
	)

	fileStorageService := filestorageservice.NewTraceDecorator(
		traceProvider,
		cfg.Tracing.Spans.Services.FileStorage,
		filestorageservice.New(
			filestoragerepository.NewTraceDecorator(
				traceProvider,
				cfg.Tracing.Spans.Repositories.FileStorage,
				filestoragerepository.New(cfg.FileStorage.BasePath, logger),
			),
		),
	)

	fileStorageUseCases := filestorageusecases.NewTraceDecorator(
		traceProvider,
		cfg.Tracing.Spans.UseCases.FileStorage,
		filestorageusecases.New(fileStorageService, cfg.FileStorage),
	)

	usersUseCases := usersusecases.NewTraceDecorator(
		traceProvider,
		cfg.Tracing.Spans.UseCases.Users,
		usersusecases.New(
			usersService,
			fileStorageService,
			cfg.Security,
			cfg.Validation,
			cfg.FileStorage,
		),
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

	notificationsUseCases := notificaionsusecases.NewTraceDecorator(
		traceProvider,
		cfg.Tracing.Spans.UseCases.Notifications,
		notificaionsusecases.New(
			notificationsService,
			usersService,
			messagesService,
			chatsService,
			webPushSubscriptionsService,
			logger,
		),
	)

	webPushSubscriptionsUseCases := webpushsubscriptionsusecases.NewTraceDecorator(
		traceProvider,
		cfg.Tracing.Spans.UseCases.WebPushSubscriptions,
		webpushsubscriptionsusecases.New(
			webPushSubscriptionsService,
		),
	)

	emailNotificationWorker, err := customnats.NewConsumer(
		cfg.NATS.ClientURL,
		cfg.NATS.Subjects.EmailNotification,
		customnats.WithGoroutinesPoolSize(cfg.NATS.GoroutinesPoolSize),
		customnats.WithMessageChannelBufferSize(cfg.NATS.MessageChannelBufferSize),
		customnats.WithNatsOptions(nats.Name(cfg.NATS.Workers.EmailNotification.Name)),
		customnats.WithMessageHandler(
			messagehandlerbuildertracingdecorator.New(
				traceProvider,
				cfg.Tracing.Spans.Handlers.EmailNotification,
				emailnotificationmessagehandlerbuilder.New(
					notificationsUseCases,
					settingsUseCases,
					logger,
				),
			).MessageHandler(context.Background()),
		),
	)
	if err != nil {
		panic(err)
	}

	if err = emailNotificationWorker.Run(); err != nil {
		panic(err)
	}

	defer func() {
		if err = emailNotificationWorker.Stop(); err != nil {
			logging.LogError(
				logger,
				fmt.Sprintf(
					"Error shutting down %q worker",
					cfg.NATS.Workers.EmailNotification.Name,
				),
				err,
			)
		}
	}()

	webPushNotificationWorker, err := customnats.NewConsumer(
		cfg.NATS.ClientURL,
		cfg.NATS.Subjects.WebPushNotification,
		customnats.WithGoroutinesPoolSize(cfg.NATS.GoroutinesPoolSize),
		customnats.WithMessageChannelBufferSize(cfg.NATS.MessageChannelBufferSize),
		customnats.WithNatsOptions(nats.Name(cfg.NATS.Workers.WebPushNotification.Name)),
		customnats.WithMessageHandler(
			messagehandlerbuildertracingdecorator.New(
				traceProvider,
				cfg.Tracing.Spans.Handlers.WebPushNotification,
				webpushnotificationmessagehandlerbuilder.New(
					notificationsUseCases,
					settingsUseCases,
					logger,
				),
			).MessageHandler(context.Background()),
		),
	)
	if err != nil {
		panic(err)
	}

	if err = webPushNotificationWorker.Run(); err != nil {
		panic(err)
	}

	defer func() {
		if err = webPushNotificationWorker.Stop(); err != nil {
			logging.LogError(
				logger,
				fmt.Sprintf(
					"Error shutting down %q worker",
					cfg.NATS.Workers.WebPushNotification.Name,
				),
				err,
			)
		}
	}()

	upgrader := &websocket.Upgrader{
		HandshakeTimeout: cfg.Websocket.HandshakeTimeout,
	}

	c, err := controllers.New(
		cfg.HTTP,
		cfg.CORS,
		cfg.Docs,
		cfg.Cookies,
		cfg.NATS,
		usersUseCases,
		authUseCases,
		chatsUseCases,
		messagesUseCases,
		settingsUseCases,
		webPushSubscriptionsUseCases,
		fileStorageUseCases,
		logger,
		traceProvider,
		upgrader,
		natsPublisher,
		cfg.Tracing.Spans.Root,
		cfg.Security,
		[]string{ // Чувствительная информация, которая не должна быть заллогирована
			"email",
			"password",
			"oldPassword",
			"newPassword",
		},
		cfg.WebPush.VAPIDPublicKey,
	)
	if err != nil {
		panic(err)
	}

	application := app.New(c)
	application.Run()
}
