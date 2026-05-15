# Пакет internal/errors

## Назначение

Sentinel-ошибки для каждого домена. Позволяют верхним слоям (usecases, HTTP-хэндлеры) принимать решения через `errors.Is`.

## Ошибки по файлам

### users.go — пользователи

| Ошибка | Описание |
|---|---|
| `ErrUserAlreadyExists` | Пользователь с такими данными уже зарегистрирован |
| `ErrUserNotFound` | Пользователь не найден |

### auth.go — аутентификация и авторизация

| Ошибка | Описание |
|---|---|
| `ErrEmailNotConfirmed` | Email пользователя не подтверждён |
| `ErrEmailAlreadyConfirmed` | Email уже был подтверждён ранее |
| `ErrWrongPassword` | Неверный пароль |
| `ErrAccessTokenDoesNotBelongToRefreshToken` | Access-токен не соответствует refresh-токену |
| `ErrNewPasswordEqualToOldPassword` | Новый пароль совпадает со старым |

### chats.go — чаты

| Ошибка | Описание |
|---|---|
| `ErrInvalidChat` | Чат не прошёл валидацию (тип, количество участников) |
| `ErrUserIsNotChatMember` | Пользователь не является участником чата |
| `ErrChatNotFound` | Чат не найден |
| `ErrChatAlreadyExists` | Чат между участниками уже существует |

### validation.go — валидация входных данных

| Ошибка | Описание |
|---|---|
| `ErrValidationFailed` | Данные не прошли проверку по regexp-правилам |

### security.go — безопасность / токены

| Ошибка | Описание |
|---|---|
| `ErrInvalidJWT` | JWT-токен невалиден (подпись, формат) |
| `ErrTokenExpired` | Срок действия токена истёк |

### settings.go — настройки

| Ошибка | Описание |
|---|---|
| `ErrSettingsNotFound` | Настройки пользователя не найдены |

### limit.go — ограничения

| Ошибка | Описание |
|---|---|
| `ErrLimitExceeded` | Превышен лимит попыток (rate limiting) |

### push_subscriptions.go — push-подписки

| Ошибка | Описание |
|---|---|
| `ErrPushSubscriptionNotFound` | Push-подписка не найдена |

## Зависимости

Только стандартная библиотека (`errors`).
