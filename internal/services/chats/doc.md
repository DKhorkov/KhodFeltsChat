# services/chats — Сервис чатов

## Назначение

Комбинирует `ChatsRepository` и `MessagesRepository` для операций с чатами.

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `GetUserChats` | Получение чатов пользователя + обогащение участниками и последним сообщением |
| `CreateChat` | Создание чата + bulk insert участников, возврат полного чата |
| `GetChatMembers` | Проверка существования чата + получение участников |
| `PrivateChatExists` | Проверка существования приватного чата между набором участников |

## Зависимости

- Factory functions для `ChatsRepository` и `MessagesRepository`
- `UnitOfWork`
