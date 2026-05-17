# Пакет usecases/notifications

## Назначение

Бизнес-логика для отправки email-уведомлений пользователям.

## Ключевые операции

### SendVerifyEmailMessage
- Получает пользователя по ID через `UsersService`.
- Проверяет, что email ещё **не** подтверждён (`!EmailConfirmed`).
- Вызывает `NotificationsService` для отправки письма с подтверждением.

### SendForgetPasswordMessage
- Получает пользователя по ID через `UsersService`.
- Проверяет, что email **подтверждён** (`EmailConfirmed`), иначе возвращает ошибку.
- Вызывает `NotificationsService` для отправки письма со сбросом пароля.

### SendNewMessageNotification
- Получает пользователя по ID через `UsersService`.
- Вызывает `NotificationsService` для отправки email-уведомления о новом сообщении.

## Зависимости

- `internal/interfaces` — `NotificationsService`, `UsersService`.
- `internal/domains` — `User`.
- `internal/errors` — ошибки состояния подтверждения email.
