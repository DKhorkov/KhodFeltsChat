# repositories/chats

## Назначение

Репозиторий для работы с чатами и их участниками. Обслуживает операции создания
чатов, получения списков чатов пользователя и определения статуса прочтения.

## Таблицы

- `chats` — основная таблица чатов (title, description, type)
- `chats_members` — связь чатов и участников (user_id, chat_id)

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `GetChatByID(ctx, id, userID)` | Возвращает чат по ID c рассчитанным `unread_count` для `userID` (агрегат `COUNT(*) FROM messages_statuses WHERE user_id=$userID AND is_read=false AND is_deleted=false`). Если непрочитанные для конкретного юзера не нужны (existence-чек или фетч свежесозданного чата), передаётся `userID=0` — подзапрос вернёт 0. Участники и сообщения не подтягиваются |
| `GetUserChats(ctx, userID, pagination)` | Список чатов пользователя, упорядоченных по времени последнего сообщения (`COALESCE(MAX(messages.created_at), chats.updated_at) DESC`). Для каждого чата возвращается `unread_count` — скалярный подзапрос `COUNT(*)` к `messages_statuses` (фильтры `user_id`, `is_read = false`, `is_deleted = false`). Используется фронтом для бейджа PWA и числа непрочитанных в списке чатов; статус «прочитан» определяется на клиенте как `unread_count == 0` |
| `CreateChat(ctx, chat)` | Вставляет чат в `chats`, затем пакетно (bulk) вставляет участников в `chats_members` |
| `GetChatMembers(ctx, chatID)` | JOIN `users` и `chats_members` — возвращает список участников чата |
| `PrivateChatExists(ctx, members)` | Проверяет существование приватного чата для набора участников через коррелированные EXISTS-подзапросы |

## Зависимости

- `pg.Transaction` — все запросы выполняются в рамках переданной транзакции
- `github.com/Masterminds/squirrel` — построитель SQL-запросов
- `github.com/DKhorkov/libs/logging` — логирование ошибок закрытия rows
- `github.com/DKhorkov/kfc/internal/domains` — типы `Chat`, `User`, `Pagination`, `ChatTypePrivate`

## Trace-декоратор

`TraceDecorator` оборачивает все методы репозитория трассировкой OpenTelemetry.
