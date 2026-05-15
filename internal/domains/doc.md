# Пакет internal/domains

## Назначение

Доменные модели и DTO, используемые на всех слоях приложения.

## Модели

### User
Основная сущность пользователя: `ID`, `Username`, `Email`, `EmailConfirmed`, `Password`, `CreatedAt`, `UpdatedAt`.

### Chat
Чат с типами `private` / `group`.
- Метод `IsValid() bool` — проверяет допустимость типа и количество участников:
  - минимум 1 участник для любого чата,
  - ровно 2 участника для приватного чата.
- Поля: `ID`, `Title`, `Description`, `Type`, `IsRead`, `Members []User`, `Messages []Message`.

### Message
Сообщение в чате. Реализует **паттерн Builder**:
- `NewMessage() *Message` — конструктор.
- `From(user User) *Message` — устанавливает отправителя.
- `Received() *Message` — проставляет `CreatedAt = time.Now()`.
- `Updated() *Message` — проставляет `UpdatedAt = time.Now()`.

### RefreshToken
Токен обновления сессии: `ID`, `UserID`, `Value`, `TTL`, `CreatedAt`, `UpdatedAt`.

### Pagination
Параметры постраничной выборки: `Limit *uint64`, `Offset *uint64`.

## DTO

| Тип | Назначение |
|---|---|
| `LoginDTO` | Вход: `Login`, `Password` |
| `RegisterDTO` | Регистрация: `Username`, `Email`, `Password` |
| `TokensDTO` | Пара токенов: `AccessToken`, `RefreshToken` |
| `ForgetPasswordDTO` | Новый пароль при восстановлении |
| `ChangePasswordDTO` | Смена пароля: `UserID`, `OldPassword`, `NewPassword` |
| `UpdateUserDTO` | Обновление профиля: `ID`, `Username` |
| `UsersFilters` | Фильтр по `Username` |
| `SendVerifyEmailMessageDTO` | Email для отправки письма верификации |
| `SendForgetPasswordMessageDTO` | Email для отправки письма восстановления пароля |

### WebPushSubscription
Push-подписка пользователя на Web Push уведомления: `ID`, `UserID`, `Endpoint`, `EncryptionKey`, `Auth`, `CreatedAt`.

## Notification DTO (для NATS-воркеров)

| Тип | Поля |
|---|---|
| `VerifyEmailNotificationDTO` | `UserID uint64` |
| `ForgetPasswordNotificationDTO` | `UserID uint64` |
| `WebPushNotificationDTO` | `UserID uint64`, `MessageID uint64` |

## Зависимости

Только стандартная библиотека (`time`, `slices`).
