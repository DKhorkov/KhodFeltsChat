# repositories/messages

## Назначение

Репозиторий для работы с сообщениями и их статусами прочтения. Хранит сообщения
чата и индивидуальные статусы `is_read` для каждого участника.

## Таблицы

- `messages` — текст сообщения, отправитель, чат
- `messages_statuses` — флаг `is_read` для каждой пары (message_id, user_id)

## Вспомогательный тип

`MessagePg` — плоская структура для сканирования результата JOIN-запроса
(поля сообщения + поля отправителя как `SenderID`, `SenderUsername` и т.д., + nullable reply-поля: `ReplyToMessageID`, `ReplyToMessageText`, `ReplyToMessageCreatedAt`, `ReplyToSenderID`, `ReplyToSenderUsername`).
Преобразуется в `domains.Message` функцией `pgMessageToDomainMessage` (заполняет `ReplyToMessage`, если reply-поля не nil).

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `SaveMessage(ctx, message)` | Вставляет сообщение в `messages` (включая `reply_to_message_id`), затем получает всех участников чата из `chats_members` и пакетно создаёт записи в `messages_statuses` (`is_read = true` для отправителя, `false` для остальных) |
| `GetChatMessages(ctx, userID, chatID, pagination)` | Постраничная выборка сообщений с JOIN на `users`, `messages_statuses` и LEFT JOIN на reply-сообщение + reply-отправитель; фильтр `is_deleted = false`; сортировка `id DESC` |
| `GetMessageByID(ctx, userID, messageID)` | Получение одного сообщения с read-статусом текущего пользователя и reply-данными; фильтр `is_deleted = false` |
| `ChangeMessagesIsReadStatus(ctx, userID, messages, isRead)` | Пакетное обновление `is_read` в `messages_statuses` для списка сообщений и конкретного пользователя |
| `DeleteMessageForUser(ctx, userID, messageID)` | Soft-удаление: устанавливает `is_deleted = true` в `messages_statuses` для конкретного пользователя |
| `DeleteMessageForAll(ctx, messageID)` | Soft-удаление для всех: устанавливает `is_deleted = true` во всех записях `messages_statuses` для данного сообщения |
| `ReadAllChatMessages(ctx, userID, chatID)` | Пакетная пометка всех непрочитанных сообщений чата как прочитанных для пользователя |
| `UpdateMessageText(ctx, messageID, text)` | Обновляет `text` и `updated_at` в таблице `messages` для указанного сообщения |

## Зависимости

- `pg.Transaction` — все запросы выполняются в рамках переданной транзакции
- `github.com/Masterminds/squirrel` — построитель SQL-запросов
- `github.com/DKhorkov/libs/logging` — логирование ошибок закрытия rows
- `github.com/DKhorkov/kfc/internal/domains` — типы `Message`, `Pagination`

## Trace-декоратор

`TraceDecorator` оборачивает все методы репозитория трассировкой OpenTelemetry.
