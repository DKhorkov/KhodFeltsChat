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
| `GetChatByID(ctx, id)` | Возвращает чат по ID (без участников и сообщений) |
| `GetUserChats(ctx, userID, pagination)` | Список чатов пользователя, упорядоченных по времени последнего сообщения (`COALESCE(MAX(messages.created_at), chats.updated_at) DESC`). Статус прочитанности (`is_read`) вычисляется через `NOT EXISTS` подзапрос к `messages_statuses` |
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
