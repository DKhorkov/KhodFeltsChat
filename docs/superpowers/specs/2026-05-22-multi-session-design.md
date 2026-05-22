# Мультисессии: одна сессия на каждое устройство

**Дата:** 2026-05-22
**Статус:** Согласован

## Проблема

Сейчас у пользователя хранится один refresh token. При логине с нового устройства старый токен удаляется — пользователя "выкидывает" с первого устройства.

## Решение

Разрешить несколько refresh-токенов на одного пользователя. Каждый логин создаёт новую сессию, не трогая существующие. Устройство определяется тем, какой refresh token лежит в cookies конкретного браузера.

- Без явной идентификации устройства (User-Agent, fingerprint)
- Без лимита на количество сессий — TTL подчищает протухшие токены
- Безопасность не страдает: refresh token защищён (JWT + HttpOnly cookie), а для его получения нужны валидные credentials

## Схема БД

Изменений не требуется. Таблица `refresh_tokens` уже поддерживает несколько строк на одного `user_id` — UNIQUE constraint стоит только на `value`. Индекс на `value` создаётся автоматически из UNIQUE constraint.

## Слой репозитория (`AuthRepository`)

### Удаляется

- `GetRefreshTokenByUserID(ctx, userID)` — больше не нужен

### Добавляется

- `GetRefreshTokenByValue(ctx, value string) (*domains.RefreshToken, error)` — поиск токена по значению. SQL: `SELECT * FROM refresh_tokens WHERE value = $1 AND ttl > CURRENT_TIMESTAMP`
- `ExpireAllUserRefreshTokens(ctx, userID uint64) error` — удаление всех токенов пользователя. SQL: `DELETE FROM refresh_tokens WHERE user_id = $1`

### Без изменений

- `CreateRefreshToken` — создаёт токен (теперь их может быть несколько на одного пользователя)
- `ExpireRefreshToken` — удаляет конкретный токен по ID

## Слой сервисов (`AuthService`)

Зеркалит изменения репозитория:

### Удаляется

- `GetRefreshTokenByUserID`

### Добавляется

- `GetRefreshTokenByValue(ctx, value string) (*domains.RefreshToken, error)`
- `ExpireAllUserRefreshTokens(ctx, userID uint64) error`

### Изменения в существующих методах

- `ForgetPassword` — заменить `GetRefreshTokenByUserID` + `ExpireRefreshToken` на `ExpireAllUserRefreshTokens`. При сбросе пароля разлогиниваем со всех устройств.

## Слой usecases (`AuthUseCases`)

### `LoginUser`

Убираем блок удаления старого refresh token:

```go
// Удалить:
dbRefreshToken, err := u.authService.GetRefreshTokenByUserID(ctx, user.ID)
if err == nil {
    if err = u.authService.ExpireRefreshToken(ctx, dbRefreshToken.ID); err != nil {
        return nil, err
    }
}
```

Просто создаём новый токен, не трогая существующие сессии.

### `RefreshTokens`

Упрощаем флоу:

```
Было:  decode refresh → parse JWT → get access token → parse JWT → get userID → find by userID → compare values
Стало: decode refresh → find by value → get userID from DB record
```

Связка access↔refresh убирается. Access token самодостаточен (содержит userID, подписан, имеет TTL). Refresh token нужен только для выпуска новой пары. Принадлежность гарантируется записью в БД.

### `LogoutUser`

Меняется сигнатура:

```go
// Было:
LogoutUser(ctx context.Context, userID uint64) error

// Стало:
LogoutUser(ctx context.Context, refreshToken string) error
```

Декодирует refresh token → находит по значению в БД → удаляет конкретную сессию.

### `LogoutUserFromAllSessions` (новый)

```go
LogoutUserFromAllSessions(ctx context.Context, userID uint64) error
```

Вызывает `ExpireAllUserRefreshTokens(userID)`.

## Слой контроллеров (HTTP handlers)

### Без изменений

- `POST /api/sessions` (login) — вызывает `LoginUser`, ставит cookies
- `PUT /api/sessions` (refresh) — читает cookie, вызывает `RefreshTokens`, ставит новые cookies

### Изменения

- `DELETE /api/sessions` (logout) — теперь читает refresh token из cookie и передаёт в `LogoutUser(refreshToken)` вместо `LogoutUser(userID)`

### Новый хэндлер

- `DELETE /api/sessions/all` (logout all) — использует `userID` из контекста (auth middleware), вызывает `LogoutUserFromAllSessions(userID)`, удаляет cookies текущего браузера

### Итого по роутам

| Метод | Путь | Действие |
|-------|------|----------|
| POST | `/api/sessions` | Login (без изменений) |
| PUT | `/api/sessions` | Refresh tokens (без изменений) |
| DELETE | `/api/sessions` | Logout текущей сессии (изменён) |
| DELETE | `/api/sessions/all` | Logout всех сессий (новый) |

## Тестирование

### Repository (`repository_test.go`)

- `GetRefreshTokenByValue` — находит токен по значению
- `GetRefreshTokenByValue` — ошибка для протухшего токена
- `ExpireAllUserRefreshTokens` — удаляет все токены пользователя
- Несколько токенов на одного `user_id` корректно создаются

### Usecases (`usecases_test.go`)

- `LoginUser` — не удаляет существующие токены при повторном логине
- `RefreshTokens` — ротирует конкретный токен, не трогая остальные сессии
- `LogoutUser` — удаляет только текущую сессию по refresh token
- `LogoutUserFromAllSessions` — удаляет все сессии пользователя

### Handlers (`handler_test.go`)

- Logout handler — передаёт refresh token из cookie вместо userID
- Новый logout_all handler — базовые кейсы (success, unauthorized)

### Trace decorator тесты

- Обновить под новые сигнатуры методов
