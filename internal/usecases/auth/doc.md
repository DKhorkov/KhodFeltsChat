# Пакет usecases/auth

## Назначение

Реализует бизнес-логику аутентификации и управления учётными записями.

## Ключевые операции

### RegisterUser
- Нормализует email к нижнему регистру.
- Валидирует email, пароль и имя пользователя через regex.
- Хэширует пароль (bcrypt).
- Создаёт пользователя через сервис.

### LoginUser
- Ищет пользователя по email (в нижнем регистре); при неудаче — по username (регистр сохраняется, т.к. username case-sensitive).
- Проверяет флаг `EmailConfirmed`.
- Валидирует пароль.
- Создаёт новую сессию (refresh-токен), не удаляя существующие (мультисессионность).
- Возвращает пару JWT (access + base64-refresh).

### RefreshTokens
- Декодирует refresh-токен из base64.
- Находит токен в БД по значению, получает userID из записи.
- Ротирует пару токенов.

### LogoutUser
- Декодирует refresh-токен, находит в БД по значению.
- Инвалидирует конкретный refresh-токен (текущая сессия).

### LogoutUserFromAllSessions
- Инвалидирует все refresh-токены пользователя (все сессии).

### VerifyEmail / ForgetPassword
- Принимают `userID uint64` (код→userID уже разрешён декоратором).
- Проверяют состояние пользователя (не подтверждён / не совпадает пароль).
- Вызывают соответствующий метод сервиса.

## CacheDecorator

Декоратор поверх основного usecase:
- Rate limiting через Redis для `Send*Message`: не более 3 попыток за 3 минуты на email.
- Резолвит 6-значный код в userID через `{prefix}:{code}` в Redis, передаёт `userID` в base, удаляет ключ после успешного использования (одноразовость).

## Зависимости

- `internal/interfaces` — `AuthService`, `UsersService`.
- `internal/domains` — `User`, `AuthTokens`.
- Redis-клиент (через `internal/common/cache`).
- `golang.org/x/crypto/bcrypt`.
