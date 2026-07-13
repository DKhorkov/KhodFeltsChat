# Индекс модулей

| Пакет | Описание |
|-------|----------|
| `cmd/` | Точка входа, инициализация DI, запуск серверов и NATS workers |
| `cmd/publishers/` | Тестовые утилиты для ручной публикации NATS сообщений |
| `internal/app/` | Обёртка запуска приложения с graceful shutdown (SIGINT/SIGTERM) |
| `internal/config/` | Конфигурация из переменных окружения со значениями по умолчанию |
| `internal/domains/` | Доменные модели (User, Chat, Message, RefreshToken) и DTO |
| `internal/common/` | Общие константы: пути логов, формат дат, timezone, настройки кэша |
| `internal/errors/` | Sentinel ошибки по доменам (users, auth, chats, validation, security, limit) |
| `internal/interfaces/` | Все интерфейсы с mockgen директивами |
| `internal/repositories/auth/` | Репозиторий авторизации: register, tokens, verify email, change password |
| `internal/repositories/users/` | CRUD пользователей с фильтрацией и пагинацией |
| `internal/repositories/chats/` | Чаты, участники, проверка существования приватного чата |
| `internal/repositories/messages/` | Сообщения, статусы прочтения, bulk insert статусов |
| `internal/repositories/reactions/` | Справочник emoji + M2M реакций юзер↔сообщение |
| `internal/repositories/emails/` | SMTP отправка email через gomail (не работает с БД) |
| `internal/services/auth/` | Бизнес-логика авторизации, дедупликация, NATS публикация событий |
| `internal/services/users/` | CRUD сервис пользователей через UoW |
| `internal/services/chats/` | Сервис чатов: обогащение участниками и последним сообщением |
| `internal/services/messages/` | Сервис сообщений: сохранение + управление статусом прочтения |
| `internal/services/reactions/` | Сервис реакций (UoW-обёртка над ReactionsRepository) |
| `internal/services/notifications/` | Тонкий фасад над EmailsRepository |
| `internal/usecases/auth/` | Юзкейсы авторизации + CacheDecorator (rate limit, token validation) |
| `internal/usecases/users/` | Юзкейсы пользователей с валидацией username |
| `internal/usecases/chats/` | Юзкейсы чатов: валидация, проверка участников, дедупликация |
| `internal/usecases/messages/` | Юзкейсы сообщений: проверка membership перед доступом + приватный attachReactions через reactionsService |
| `internal/usecases/reactions/` | Юзкейсы реакций: валидация member/reaction, AddReaction возвращает `*domains.Reaction` (WS-фан-аут делает HTTP handler через WSBroadcaster) |
| `internal/usecases/notifications/` | Юзкейсы уведомлений: проверка emailConfirmed |
| `internal/controllers/http/` | HTTP контроллер: gorilla/mux роутер, 5 middleware, graceful shutdown |
| `internal/controllers/http/handlers/` | Все HTTP обработчики (auth, users, chats, messages, ws, docs) |
| `internal/controllers/http/schemas/` | Swagger-аннотированные request/response структуры |
| `internal/controllers/http/mappers/` | Преобразование domain моделей в schema (MapUser, MapChat, MapMessage) |
| `internal/uow/` | Unit of Work: PostgreSQL транзакции с goroutine execution + trace decorator |
| `internal/workers/` | NATS consumer workers: verify-email, forget-password + tracing decorator |
| `internal/contentbuilders/` | Генерация email HTML + сохранение одноразовых токенов в Redis |
| `mocks/` | Сгенерированные mockgen моки для всех интерфейсов |
| `migrations/` | Goose SQL миграции (users, refresh_tokens, chats, messages) |
| `build/` | Docker: Dockerfile, docker-compose (local/prod), backup cron |
| `api/` | OpenAPI/Swagger спецификация (swagger.yaml) |
| `scripts/` | Taskfile.yml (команды сборки/запуска), скрипты PostgreSQL |
