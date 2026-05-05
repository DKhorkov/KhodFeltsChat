# migrations — SQL миграции

## Назначение

Goose SQL миграции для PostgreSQL.

## Миграции

| Файл | Описание |
|------|----------|
| `20240919172036_users_table.sql` | Таблица `users` (id, username, email, email_confirmed, password, created_at, updated_at) |
| `20241015183033_refresh_tokens_table.sql` | Таблица `refresh_tokens` (id, user_id, value, expires_at, created_at, updated_at) |
| `20260106113434_chats.sql` | Таблицы `chats` и `chats_members` |
| `20260330171139_message_is_read.sql` | Таблица `messages_statuses` (статусы прочтения сообщений) |

## Команды

```bash
task migrate-up      # Применить миграции
task migrate-down    # Откатить последнюю
```
