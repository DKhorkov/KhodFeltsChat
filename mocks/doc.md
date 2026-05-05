# mocks — Сгенерированные моки

## Назначение

Моки всех интерфейсов, сгенерированные через `go.uber.org/mock/mockgen`.

## Содержимое

| Директория | Моки для |
|-----------|----------|
| `repositories/` | UsersRepository, AuthRepository, ChatsRepository, MessagesRepository (MessagesService), EmailsRepository |
| `services/` | UsersService, AuthService, ChatsService, MessagesService, NotificationsService |
| `usecases/` | UsersUseCases, AuthUseCases, ChatsUseCases, MessagesUseCases, NotificationsUseCases |
| `controllers/` | Controller |
| `uow/` | UnitOfWork |
| `upgrader/` | Upgrader (WebSocket) |
| `workers/` | MessageHandlerBuilder |
| `contentbuilders/` | VerifyEmailContentBuilder, ForgetPasswordContentBuilder |

## Генерация

Моки генерируются на основе `//go:generate mockgen` директив в `internal/interfaces/`.
