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
| `20260513000000_settings.sql` | Таблица `settings` (пользовательские настройки) |
| `20260515000000_web_push_subscriptions.sql` | Таблица `push_subscriptions` (endpoint, encryption_key, auth, user_id) |
| `20260520000000_remove_chats_members_is_read.sql` | Удаление колонки `is_read` из `chats_members` (статус прочитанности вычисляется из `messages_statuses`) |

## Команды

```bash
task migrate-up      # Применить миграции
task migrate-down    # Откатить последнюю
```
