# services/messages — Сервис сообщений

## Назначение

Управление сообщениями и статусами прочтения.

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `SaveMessage` | Сохранение сообщения → re-fetch созданного сообщения → пометка всех сообщений чата прочитанными для отправителя |
| `GetChatMessages` | Проверка существования чата → paginated fetch → пометка полученных сообщений как прочитанных для запрашивающего пользователя (через `ChangeMessagesIsReadStatus`) |
| `GetMessageByID` | Получение одного сообщения по ID в контексте пользователя (через `MessagesRepository`) |
| `DeleteMessage` | Удаление сообщения: если `ForAll` — вызывает `DeleteMessageForAll`, иначе `DeleteMessageForUser` |

## Зависимости

- Factory functions для `MessagesRepository` и `ChatsRepository`
- `UnitOfWork`
