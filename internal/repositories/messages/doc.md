# repositories/messages

## Назначение

Репозиторий для работы с сообщениями и их статусами прочтения. Хранит сообщения
чата и индивидуальные статусы `is_read` для каждого участника.

## Таблицы

- `messages` — текст сообщения, отправитель, чат
- `messages_statuses` — флаг `is_read` для каждой пары (message_id, user_id)

## Вспомогательный тип

`MessagePg` — плоская структура для сканирования результата JOIN-запроса
(поля сообщения + поля отправителя как `SenderID`, `SenderUsername` и т.д.).
Преобразуется в `domains.Message` функцией `pgMessageToDomainMessage`.

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `SaveMessage(ctx, message)` | Вставляет сообщение в `messages`, затем получает всех участников чата из `chats_members` и пакетно создаёт записи в `messages_statuses` (`is_read = true` для отправителя, `false` для остальных) |
| `GetChatMessages(ctx, userID, chatID, pagination)` | Постраничная выборка сообщений с JOIN на `users` и `messages_statuses` (фильтр по `user_id`), сортировка `id DESC` |
| `GetMessageByID(ctx, userID, messageID)` | Получение одного сообщения с read-статусом текущего пользователя |
| `ChangeMessagesIsReadStatus(ctx, userID, messages, isRead)` | Пакетное обновление `is_read` в `messages_statuses` для списка сообщений и конкретного пользователя |

## Зависимости

- `pg.Transaction` — все запросы выполняются в рамках переданной транзакции
- `github.com/Masterminds/squirrel` — построитель SQL-запросов
- `github.com/DKhorkov/libs/logging` — логирование ошибок закрытия rows
- `github.com/DKhorkov/kfc/internal/domains` — типы `Message`, `Pagination`

## Trace-декоратор

`TraceDecorator` оборачивает все методы репозитория трассировкой OpenTelemetry.
