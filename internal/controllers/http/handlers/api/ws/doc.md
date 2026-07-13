# Пакет handlers/api/ws

## Назначение

WebSocket-обработчик для real-time обмена сообщениями.

## Маршрут

`GET /api/ws` → `Handle(w, r)` — Upgrade до WebSocket с cookie-аутентификацией.

## Мультисессия

Один пользователь может иметь несколько WebSocket-соединений (разные устройства/вкладки). Соединения хранятся в `sync.Map[userID → *userConnections]`, где `userConnections` содержит `sync.Mutex` + `[]*websocket.Conn`.

## Жизненный цикл соединения

1. Аутентификация через `UserIDContextKey` из контекста
2. Получение пользователя через `usersUseCases.GetUserByID`
3. Upgrade соединения через `upgrader.Upgrade`
4. `addConnection(userID, conn)` — добавление в пул
5. `listen(conn, user)` — цикл чтения входящих сообщений
6. `removeConnection(userID, conn)` — удаление при закрытии

## Входящие сообщения

`listen()` читает JSON из WebSocket, валидирует участие отправителя в чате, сохраняет сообщение через `messagesUseCases.SaveMessage`, оборачивает в `WSEvent` (`type: "new_message"`) и рассылает всем онлайн-участникам чата (включая отправителя). Для офлайн-участников публикует уведомления в NATS (web-push и email).

## WSBroadcaster

Реализует интерфейс `interfaces.WSBroadcaster`:

| Метод | Описание |
|-------|----------|
| `BroadcastMessageDeleted(ctx, chatID, messageID)` | Рассылает `message_deleted` событие всем участникам чата (удаление у всех) |
| `SendMessageDeletedToUser(ctx, chatID, messageID, userID)` | Отправляет `message_deleted` событие только конкретному пользователю (удаление у себя) |
| `BroadcastMessageEdited(ctx, chatID, messageID, text)` | Рассылает `message_edited` событие всем участникам чата (редактирование сообщения) |
| `BroadcastReactionAdded(ctx, messageID, userID, reactionID)` | Рассылает `reaction_added`. chatID резолвит сам через `messagesUseCases.GetMessageByID` — usecase-слой реакций не тянет транспортные детали. Emoji в payload не шлётся: клиент лукапит его в справочнике `/api/reactions` |
| `BroadcastReactionRemoved(ctx, messageID, userID, reactionID)` | Рассылает `reaction_removed`. То же с chatID |

## Вспомогательные методы

| Метод | Описание |
|-------|----------|
| `addConnection(userID, conn)` | Добавляет соединение в пул пользователя |
| `removeConnection(userID, conn)` | Удаляет соединение; при пустом пуле — удаляет запись из `sync.Map` |
| `sendToUser(ctx, userID, event)` | Отправляет событие во все соединения пользователя; битые соединения закрываются и удаляются |
| `hasConnections(userID)` | Проверяет наличие активных соединений у пользователя |

## Зависимости

- `interfaces.Upgrader` — WebSocket upgrade
- `interfaces.UsersUseCases` — получение пользователя
- `interfaces.ChatsUseCases` — получение участников чата
- `interfaces.MessagesUseCases` — сохранение сообщений
- `customnats.Publisher` — публикация уведомлений
- `config.NATSConfig` — NATS subjects
