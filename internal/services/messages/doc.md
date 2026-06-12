# services/messages — Сервис сообщений

## Назначение

Управление сообщениями и статусами прочтения.

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `SaveMessage` | Сохранение сообщения → re-fetch созданного сообщения → пометка всех сообщений чата прочитанными для отправителя |
| `GetChatMessages` | Проверка существования чата → paginated fetch → пометка полученных сообщений как прочитанных для запрашивающего пользователя (через `ChangeMessagesIsReadStatus`) |
| `GetMessageByID` | Получение одного сообщения по ID в контексте пользователя (через `MessagesRepository`) |
| `GetUserUnreadCount` | Возвращает суммарное число непрочитанных и неудалённых сообщений пользователя по всем чатам. Используется usecase'ом push-уведомлений для проставления `unreadCount` в push payload |
| `DeleteMessage` | Удаление сообщения: если `ForAll` — вызывает `DeleteMessageForAll`, иначе `DeleteMessageForUser` |
| `UpdateMessage` | Редактирование текста сообщения: обновляет `text` и `updated_at` → re-fetch обновлённого сообщения (в рамках UoW) |

## Зависимости

- Factory functions для `MessagesRepository` и `ChatsRepository`
- `UnitOfWork`
