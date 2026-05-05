# repositories/chats

## Назначение

Репозиторий для работы с чатами и их участниками. Обслуживает операции создания
чатов, получения списков чатов пользователя и управления статусом прочтения.

## Таблицы

- `chats` — основная таблица чатов (title, description, type)
- `chats_members` — связь чатов и участников, хранит флаг `is_read` для каждого участника

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `GetChatByID(ctx, id)` | Возвращает чат по ID (без участников и сообщений) |
| `GetUserChats(ctx, userID, pagination)` | Список чатов пользователя, упорядоченных по времени последнего сообщения (`COALESCE(MAX(messages.created_at), chats.updated_at) DESC`) |
| `CreateChat(ctx, chat)` | Вставляет чат в `chats`, затем пакетно (bulk) вставляет участников в `chats_members` с `is_read = false` |
| `GetChatMembers(ctx, chatID)` | JOIN `users` и `chats_members` — возвращает список участников чата |
| `ChangeChatIsReadStatus(ctx, userID, chatID, isRead)` | Обновляет `is_read` в `chats_members` для конкретного участника |
| `PrivateChatExists(ctx, members)` | Проверяет существование приватного чата для набора участников через коррелированные EXISTS-подзапросы |

## Зависимости

- `pg.Transaction` — все запросы выполняются в рамках переданной транзакции
- `github.com/Masterminds/squirrel` — построитель SQL-запросов
- `github.com/DKhorkov/libs/logging` — логирование ошибок закрытия rows
- `github.com/DKhorkov/kfc/internal/domains` — типы `Chat`, `User`, `Pagination`, `ChatTypePrivate`

## Trace-декоратор

`TraceDecorator` оборачивает все методы репозитория трассировкой OpenTelemetry.
