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
- Поле `ReplyToMessage *Message` — ссылка на сообщение, на которое отвечают (nil, если не ответ).

### WSEvent (`ws_event.go`)
WebSocket-событие (envelope для передачи типизированных данных через WS).
- `WSEventType` (string) — тип события.
  - `WSEventNewMessage` = `"new_message"` — новое сообщение.
  - `WSEventMessageDeleted` = `"message_deleted"` — удалённое сообщение.
- `WSEvent` — `Type WSEventType`, `Payload any`.
- `MessageDeletedPayload` — `MessageID uint64`, `ChatID uint64`.

### RefreshToken
Токен обновления сессии: `ID`, `UserID`, `Value`, `TTL`, `CreatedAt`, `UpdatedAt`.

### Pagination
Параметры постраничной выборки: `Limit *uint64`, `Offset *uint64`.

### Settings
Настройки пользователя: `UserID`, `Theme`, `EmailConsents`, `WebPushConsents`.
- `NotificationConsent` — битовая маска согласий (`uint64`).
- `ConsentNewMessage = 1 << iota` — согласие на уведомления о новых сообщениях.
- `HasConsent(consents, consent) bool` — проверка наличия конкретного согласия в маске.

### WebPushSubscription
Push-подписка пользователя на Web Push уведомления: `ID`, `UserID`, `Endpoint`, `EncryptionKey`, `Auth`, `CreatedAt`.

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
| `DeleteMessageDTO` | Удаление сообщения: `MessageID`, `UserID`, `ForAll` |


## Типы уведомлений (notifications.go)

| Тип | Описание |
|---|---|
| `EmailNotificationType` | Тип email-уведомления (строка): `VerifyEmail`, `ForgetPassword`, `NewMessage` |
| `WebPushNotificationType` | Тип web-push-уведомления (строка): `NewMessage` |

## Notification DTO (для NATS-воркеров)

| Тип | Поля |
|---|---|
| `EmailNotificationDTO` | `Type`, `UserID`, `Payload json.RawMessage` |
| `WebPushNotificationDTO` | `Type`, `UserID`, `Payload json.RawMessage` |
| `NewMessagePayload` | `MessageID`, `ChatID` — полезная нагрузка уведомления о новом сообщении |

Оба DTO используют двухэтапную десериализацию: сначала envelope (Type + UserID), затем Payload в зависимости от Type.

## Зависимости

Стандартная библиотека (`time`, `encoding/json`).
