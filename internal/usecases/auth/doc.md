# Пакет usecases/auth

## Назначение

Реализует бизнес-логику аутентификации и управления учётными записями.

## Ключевые операции

### RegisterUser
- Валидирует email, пароль и имя пользователя через regex.
- Хэширует пароль (bcrypt).
- Создаёт пользователя через сервис.

### LoginUser
- Ищет пользователя по email; при неудаче — по username.
- Проверяет флаг `EmailConfirmed`.
- Валидирует пароль.
- Ротирует refresh-токен, возвращает пару JWT (access + base64-refresh).

### RefreshTokens
- Декодирует refresh-токен из base64.
- Извлекает access-токен, проверяет запись в БД.
- Ротирует пару токенов.

### LogoutUser
- Инвалидирует refresh-токен (устанавливает срок истечения).

### VerifyEmail / ForgetPassword
- Декодируют base64-токен формата `salt:userID`.
- Валидируют токен через сервис.
- Вызывают соответствующий метод сервиса.

## CacheDecorator

Декоратор поверх основного usecase:
- Rate limiting через Redis: не более 3 попыток за 3 минуты на пользователя.
- Одноразовая валидация токенов через кэш (токен удаляется после первого использования).

## Зависимости

- `internal/interfaces` — `AuthService`, `UsersService`.
- `internal/domains` — `User`, `AuthTokens`.
- Redis-клиент (через `internal/common/cache`).
- `golang-jwt/jwt`, `golang.org/x/crypto/bcrypt`.
