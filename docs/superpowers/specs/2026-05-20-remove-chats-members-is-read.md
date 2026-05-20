# Удаление `chats_members.is_read` — Design Spec

## Overview

Удалить денормализованный флаг `is_read` из таблицы `chats_members` и вычислять прочитанность чата для пользователя на основе `messages_statuses.is_read`.

## Текущее состояние

Прочитанность чата хранится в двух местах:

1. **`chats_members.is_read`** — флаг на уровне чата для каждого участника
2. **`messages_statuses.is_read`** — флаг на уровне каждого сообщения для каждого участника

Синхронизация между ними происходит в `services/messages/service.go`:
- `SaveMessage`: при сохранении сообщения ставит `chats_members.is_read = false` всем участникам кроме отправителя
- `GetChatMessages`: при получении сообщений ставит `chats_members.is_read = true` для текущего пользователя

Проблема: два источника правды могут рассинхрониться (например, при ручном изменении `messages_statuses` или при сбое между двумя UPDATE).

## Целевое состояние

### 1. База данных

Миграция — удаление столбца:

```sql
ALTER TABLE chats_members DROP COLUMN is_read;
```

### 2. Вычисление прочитанности

Чат **прочитан** для пользователя, если нет ни одного сообщения в чате с `messages_statuses.is_read = false` для этого пользователя:

```sql
NOT EXISTS (
    SELECT 1 FROM messages_statuses ms
    JOIN messages m ON ms.message_id = m.id
    WHERE m.chat_id = chats.id
      AND ms.user_id = <userID>
      AND ms.is_read = false
) AS is_read
```

Для пустых чатов (без сообщений) `NOT EXISTS` возвращает `true` — чат считается прочитанным.

### 3. Удаляемый код

- Метод `ChangeChatIsReadStatus` из `ChatsRepository` интерфейса, репозитория и trace decorator
- Вызовы `ChangeChatIsReadStatus` из `services/messages/service.go` (SaveMessage и GetChatMessages)
- Колонка `is_read` из INSERT в `CreateChat`

### 4. Домен

Поле `IsRead` в `domains.Chat` **остаётся** — оно нужно для API и фронтенда. Заполняется через подзапрос в `GetUserChats`.
