# repositories/auth

## Назначение

Репозиторий для операций аутентификации и авторизации: регистрация пользователей,
управление refresh-токенами, верификация email, смена пароля.

## Таблицы

- `users` — данные пользователей
- `refresh_tokens` — refresh-токены с TTL

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `RegisterUser(ctx, RegisterDTO)` | Вставляет нового пользователя в `users`, возвращает `id` |
| `CreateRefreshToken(ctx, userID, value, ttl)` | Создаёт запись в `refresh_tokens` с вычисленным временем истечения |
| `GetRefreshTokenByUserID(ctx, userID)` | Возвращает актуальный токен (TTL > CURRENT_TIMESTAMP) |
| `ExpireRefreshToken(ctx, refreshTokenID)` | Удаляет токен из таблицы (одноразовость) |
| `VerifyEmail(ctx, userID)` | Устанавливает `email_confirmed = true` для пользователя |
| `ChangePassword(ctx, userID, newPassword)` | Обновляет поле `password` пользователя |

## Зависимости

- `pg.Transaction` — все запросы выполняются в рамках переданной транзакции
- `github.com/Masterminds/squirrel` — построитель SQL-запросов
- `github.com/DKhorkov/kfc/internal/domains` — типы `RegisterDTO`, `RefreshToken`

## Trace-декоратор

`TraceDecorator` оборачивает все методы репозитория трассировкой OpenTelemetry.
