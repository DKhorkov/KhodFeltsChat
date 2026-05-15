# services/messages — Сервис сообщений

## Назначение

Управление сообщениями и статусами прочтения.

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `SaveMessage` | Сохранение сообщения → re-fetch → обновление `is_read` для всех участников чата (sender=true, остальные=false) |
| `GetChatMessages` | Проверка существования чата → paginated fetch → пометка чата и всех полученных сообщений как прочитанных для запрашивающего пользователя |
| `GetMessageByID` | Получение одного сообщения по ID в контексте пользователя (через `MessagesRepository`) |

## Зависимости

- Factory functions для `MessagesRepository` и `ChatsRepository`
- `UnitOfWork`
