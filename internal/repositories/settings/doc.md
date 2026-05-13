# repositories/settings

## Назначение

Репозиторий для работы с пользовательскими настройками. Обслуживает операции
создания, получения и обновления настроек.

## Таблица

- `settings` — настройки пользователя (theme), связана с `users` через `user_id` (UNIQUE)

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `CreateSettings(ctx, settings)` | Вставляет запись настроек для пользователя |
| `GetSettingsByUserID(ctx, userID)` | Возвращает настройки пользователя по его ID |
| `UpdateSettings(ctx, settings)` | Обновляет тему и `updated_at` для пользователя |

## Зависимости

- `pg.Transaction` — все запросы выполняются в рамках переданной транзакции
- `github.com/Masterminds/squirrel` — построитель SQL-запросов
- `github.com/DKhorkov/kfc/internal/domains` — тип `Settings`, `ThemeType`

## Trace-декоратор

`TraceDecorator` оборачивает все методы репозитория трассировкой OpenTelemetry.
