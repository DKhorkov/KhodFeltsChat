# interfaces — Интерфейсы

Все интерфейсы проекта с `mockgen` директивами для генерации моков.

## Содержимое

### Инфраструктура
- **Controller** — `Run()`, `Stop()` — управление HTTP сервером
- **UnitOfWork** — `Do(ctx, fn(ctx, tx) error) error` — транзакционная обёртка
- **Upgrader** — `Upgrade(w, r, header) (*websocket.Conn, error)` — WebSocket upgrade
- **WSBroadcaster** — `BroadcastMessageDeleted(ctx, chatID, messageID, senderID)` — рассылка WS-события удаления сообщения участникам чата

### Repositories
- **UsersRepository** — CRUD пользователей
- **AuthRepository** — регистрация, токены (мультисессионность: GetRefreshTokenByValue, ExpireAllUserRefreshTokens), verify email, change password
- **ChatsRepository** — чаты, участники, is_read статусы, `GetChatByID`
- **MessagesRepository** — сообщения, статусы прочтения, soft-удаление (`DeleteMessageForUser`, `DeleteMessageForAll`)
- **EmailsRepository** — SMTP отправка: `SendVerifyEmailMessage(user)`, `SendForgetPasswordMessage(user)`, `SendNewMessageEmail(recipient, message, chat)` — принимает доменные объекты
- **SettingsRepository** — настройки пользователя (CRUD)
- **WebPushSubscriptionsRepository** — подписки на push-уведомления (CRUD)

### Services
- **UsersService**, **AuthService**, **MessagesService** (включая `DeleteMessage`) — бизнес-логика
- **ChatsService** — чаты + `GetChatByID`
- **NotificationsService** — email (`SendVerifyEmailMessage`, `SendForgetPasswordMessage`, `SendNewMessageEmail`) + web push (`SendWebPushNotification(subscription, message)`)
- **SettingsService** — настройки пользователя
- **WebPushSubscriptionsService** — CRUD push-подписок

### Use Cases
- **UsersUseCases**, **AuthUseCases**, **ChatsUseCases** — верхний уровень
- **MessagesUseCases** (embeds MessagesService) — сообщения + save через WS
- **NotificationsUseCases** — уведомления с явным разделением по каналам: `SendNewMessageByEmail(userID, payload)`, `SendNewMessageByWebPush(userID, payload)`, `SendVerifyEmailMessage(userID)`, `SendForgetPasswordMessage(userID)`
- **SettingsUseCases** — настройки пользователя
- **WebPushSubscriptionsUseCases** — чистый CRUD подписок: `CreateWebPushSubscription`, `DeleteWebPushSubscription`

### Workers
- **MessageHandler** — `func(msg *nats.Msg)` — обработчик NATS сообщений
- **MessageHandlerBuilder** — фабрика для создания handler с контекстом

### Content Builders
- **VerifyEmailContentBuilder** — `Subject()` + `Body(ctx, user)` для email подтверждения
- **ForgetPasswordContentBuilder** — то же для сброса пароля
- **NewMessageContentBuilder** — `Subject()` + `Body(ctx, message, chat)` — принимает доменные объекты Message и Chat
- **ContentBuilders** — агрегат всех content builders
