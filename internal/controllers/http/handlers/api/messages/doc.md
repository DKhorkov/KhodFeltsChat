# Пакет handlers/api/messages

## Назначение

Группирующая директория для обработчиков сообщений.

## Подпакеты

| Пакет | Метод | Путь | Описание |
|-------|-------|------|----------|
| `chat_messages` | GET | `/api/chats/{id}/messages` | Сообщения чата с пагинацией |
| `delete` | DELETE | `/api/messages/{id}` | Удаление сообщения (у себя или у всех) |
| `update` | PUT | `/api/messages/{id}` | Редактирование текста сообщения (только автор) |

## Зависимости

Все подпакеты используют `interfaces.MessagesUseCases`. `delete` и `update` дополнительно принимают `interfaces.WSBroadcaster`.
