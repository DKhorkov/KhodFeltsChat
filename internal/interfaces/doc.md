# interfaces — Интерфейсы

Все интерфейсы проекта с `mockgen` директивами для генерации моков.

## Содержимое

### Инфраструктура
- **Controller** — `Run()`, `Stop()` — управление HTTP сервером
- **UnitOfWork** — `Do(ctx, fn(ctx, tx) error) error` — транзакционная обёртка
- **Upgrader** — `Upgrade(w, r, header) (*websocket.Conn, error)` — WebSocket upgrade

### Repositories
- **UsersRepository** — CRUD пользователей
- **AuthRepository** — регистрация, токены, verify email, change password
- **ChatsRepository** — чаты, участники, is_read статусы
- **MessagesRepository** — сообщения, статусы прочтения
- **EmailsRepository** — SMTP отправка

### Services
- **UsersService**, **AuthService**, **ChatsService**, **MessagesService** — бизнес-логика
- **NotificationsService** (embeds EmailsRepository) — уведомления

### Use Cases
- **UsersUseCases**, **AuthUseCases**, **ChatsUseCases** — верхний уровень
- **MessagesUseCases** (embeds MessagesService) — сообщения + save через WS
- **NotificationsUseCases** — уведомления по email

### Workers
- **MessageHandler** — `func(msg *nats.Msg)` — обработчик NATS сообщений
- **MessageHandlerBuilder** — фабрика для создания handler с контекстом

### Content Builders
- **VerifyEmailContentBuilder** — `Subject()` + `Body(ctx, user)` для email подтверждения
- **ForgetPasswordContentBuilder** — то же для сброса пароля
