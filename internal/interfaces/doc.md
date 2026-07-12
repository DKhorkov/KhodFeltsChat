# interfaces — Интерфейсы

Все интерфейсы проекта с `mockgen` директивами для генерации моков.

## Содержимое

### Инфраструктура
- **Controller** — `Run()`, `Stop()` — управление HTTP сервером
- **UnitOfWork** — `Do(ctx, fn(ctx, tx) error) error` — транзакционная обёртка
- **Upgrader** — `Upgrade(w, r, header) (*websocket.Conn, error)` — WebSocket upgrade
- **WSBroadcaster** — `BroadcastMessageDeleted(ctx, chatID, messageID)` — рассылка WS-события удаления сообщения всем участникам чата; `SendMessageDeletedToUser(ctx, chatID, messageID, userID)` — отправка события удаления только конкретному пользователю (удаление у себя); `BroadcastMessageEdited(ctx, chatID, messageID, text)` — рассылка WS-события редактирования сообщения всем участникам чата; `BroadcastReactionAdded(ctx, chatID, messageID, userID, reactionID, emoji)` / `BroadcastReactionRemoved(ctx, chatID, messageID, userID, reactionID)` — рассылка WS-событий постановки/снятия реакции

### Repositories
- **FileStorageRepository** — работа с локальным хранилищем файлов: `SaveFile(ctx, filename, data) error`, `DeleteFile(ctx, filename) error`, `GetFile(ctx, filename) ([]byte, error)`
- **UsersRepository** — CRUD пользователей
- **AuthRepository** — регистрация, токены (мультисессионность: GetRefreshTokenByValue, ExpireAllUserRefreshTokens), verify email, change password
- **ChatsRepository** — чаты, участники, is_read статусы, `GetChatByID`
- **MessagesRepository** — сообщения, статусы прочтения, soft-удаление (`DeleteMessageForUser`, `DeleteMessageForAll`), редактирование текста (`UpdateMessageText`)
- **ReactionsRepository** — справочник emoji (`ListReactions`, `GetReactionByID`), M2M юзер↔сообщение↔реакция (`AddMessageReaction`, `RemoveMessageReaction`), пачечная подгрузка (`ListReactionsForMessages`)
- **EmailsRepository** — SMTP отправка: `SendVerifyEmailMessage(user)`, `SendForgetPasswordMessage(user)`, `SendNewMessageEmail(recipient, message, chat)` — принимает доменные объекты
- **SettingsRepository** — настройки пользователя (CRUD)
- **WebPushSubscriptionsRepository** — подписки на push-уведомления (CRUD)

### Services
- **FileStorageService** — сервис хранилища файлов: `SaveFile`, `DeleteFile`, `GetFile`; оборачивает `FileStorageRepository` и применяет валидацию формата/размера
- **UsersService**, **AuthService**, **MessagesService** (включая `DeleteMessage`, `UpdateMessage`) — бизнес-логика
- **ReactionsService** — UoW-обёртка над `ReactionsRepository` (та же поверхность методов)
- **ChatsService** — чаты + `GetChatByID`
- **NotificationsService** — email (`SendVerifyEmailMessage`, `SendForgetPasswordMessage`, `SendNewMessageEmail`) + web push (`SendWebPushNotification(subscription, message)`)
- **SettingsService** — настройки пользователя
- **WebPushSubscriptionsService** — CRUD push-подписок

### Use Cases
- **UsersUseCases** — верхний уровень; дополнительно содержит методы `UpdateAvatar(ctx, userID, imageData) error` и `DeleteAvatar(ctx, userID) error` для управления аватаром пользователя
- **AuthUseCases**, **ChatsUseCases** — верхний уровень
- **MessagesUseCases** (embeds MessagesService) — сообщения + save через WS
- **ReactionsUseCases** — `ListReactions`, `AddReaction(...) (chatID, emoji, err)`, `RemoveReaction(...) (chatID, err)` (broadcast делает handler), `AttachReactions([]Message)`, `AttachReaction(*Message)` — обогащение сообщений реакциями
- **NotificationsUseCases** — уведомления с явным разделением по каналам: `SendNewMessageByEmail(userID, payload)`, `SendNewMessageByWebPush(userID, payload)`, `SendVerifyEmailMessage(userID)`, `SendForgetPasswordMessage(userID)`
- **SettingsUseCases** — настройки пользователя
- **WebPushSubscriptionsUseCases** — чистый CRUD подписок: `CreateWebPushSubscription`, `DeleteWebPushSubscription`
- **FileStorageUseCases** — операции с файловым хранилищем: `Upload(ctx, path, data)`, `Download(ctx, path)`, `Delete(ctx, path)`

### Workers
- **MessageHandler** — `func(msg *nats.Msg)` — обработчик NATS сообщений
- **MessageHandlerBuilder** — фабрика для создания handler с контекстом

### Content Builders
- **VerifyEmailContentBuilder** — `Subject()` + `Body(ctx, user)` для email подтверждения
- **ForgetPasswordContentBuilder** — то же для сброса пароля
- **NewMessageContentBuilder** — `Subject()` + `Body(ctx, message, chat)` — принимает доменные объекты Message и Chat
- **ContentBuilders** — агрегат всех content builders
