# Пакет cmd

## Назначение

Точка входа в приложение. `main.go` инициализирует всю инфраструктуру и запускает сервер.

## Порядок инициализации в main.go

1. Загружаются переменные окружения (`loadenv.Init`), читается конфиг.
2. Поднимаются инфраструктурные зависимости:
   - **PostgreSQL** — пул соединений с настройками из `Config.Database`.
   - **OpenTelemetry/Jaeger** — провайдер трассировки (`tracing.New`).
   - **Redis** — провайдер кэша (`cache.New`).
   - **NATS** — publisher для отправки сообщений в очереди.
3. Строится граф зависимостей (DI вручную):
   - `UnitOfWork` → `UnitOfWork.TraceDecorator`
   - Репозитории (users, auth, chats, messages, emails) — каждый оборачивается в `TraceDecorator`.
   - Сервисы (users, auth, chats, messages, notifications) — каждый оборачивается в `TraceDecorator`.
   - UseCases (users, auth, chats, messages, notifications) — каждый оборачивается в `TraceDecorator`;
     `AuthUseCases` дополнительно оборачивается в `CacheDecorator`.
4. Запускаются два NATS-воркера:
   - **verify-email** — обрабатывает `VerifyEmailNotificationDTO`.
   - **forget-password** — обрабатывает `ForgetPasswordNotificationDTO`.
   - Каждый воркер использует `MessageHandlerBuilder` с tracing-декоратором.
5. Создаётся HTTP/WebSocket контроллер и запускается приложение через `app.New(controller).Run()`.
6. При завершении (defer) корректно закрываются: tracer, Redis, оба NATS-воркера.

## Чувствительные поля (не логируются)

`email`, `password`, `oldPassword`, `newPassword`

## cmd/publishers/

Утилиты для ручной публикации NATS-сообщений в целях тестирования:

- `cmd/publishers/verify_email/main.go` — публикует сообщение в subject `verify-email`.
- `cmd/publishers/forget_password/main.go` — публикует сообщение в subject `forget-password`.

## Зависимости

- `github.com/DKhorkov/libs/db/postgresql`
- `github.com/DKhorkov/libs/cache`
- `github.com/DKhorkov/libs/nats`
- `github.com/DKhorkov/libs/tracing`
- `github.com/DKhorkov/libs/logging`
- `github.com/gorilla/websocket`
- `github.com/nats-io/nats.go`
- Все внутренние пакеты `internal/`
