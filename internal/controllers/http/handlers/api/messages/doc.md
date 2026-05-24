# Пакет handlers/api/messages

## Назначение

Группирующая директория для обработчиков сообщений.

## Подпакеты

| Пакет | Метод | Путь | Описание |
|-------|-------|------|----------|
| `chat_messages` | GET | `/api/chats/{id}/messages` | Сообщения чата с пагинацией |
| `delete` | DELETE | `/api/messages/{id}` | Удаление сообщения (у себя или у всех) |

## Зависимости

Все подпакеты используют `interfaces.MessagesUseCases`. `delete` дополнительно принимает `interfaces.WSBroadcaster`.
