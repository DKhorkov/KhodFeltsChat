# Notification Consents System — Design Spec

## Overview

Система управления согласиями пользователя на получение уведомлений. Пользователь контролирует, какие уведомления получать (по типу события) и через какой канал (email, web push).

## Текущее состояние

- 3 NATS воркера: `verify-email`, `forget-password`, `web-push-notification`
- Каждый воркер имеет свой DTO и subject
- Push-уведомления в UI — строка со статусом текста, без серверного хранения согласий
- Таблица `settings`: `id`, `user_id`, `theme`, `created_at`, `updated_at`

## Целевое состояние

### 1. База данных

Миграция — два новых столбца в `settings`:

```sql
ALTER TABLE settings
    ADD COLUMN email_consents     INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN web_push_consents  INTEGER NOT NULL DEFAULT 0;
```

Битовая маска: бит `1` в позиции = согласие дано, `0` = не дано.

- Бит 0 (значение `1`) = уведомления о новых сообщениях
- Будущие биты: `1 << 1`, `1 << 2`, ...

Значение по умолчанию `0` — все уведомления отключены.

### 2. Домен

```go
type NotificationConsent int

const (
    ConsentNewMessage NotificationConsent = 1 << iota
    // ConsentMentions NotificationConsent = 1 << 1
    // ...
)

func HasConsent(mask, consent NotificationConsent) bool {
    return mask&consent != 0
}
```

Обновлённая модель `Settings`:

```go
type Settings struct {
    ID              uint64              `json:"id"`
    UserID          uint64              `json:"userId"`
    Theme           ThemeType           `json:"theme"`
    EmailConsents   NotificationConsent `json:"emailConsents"`
    WebPushConsents NotificationConsent `json:"webPushConsents"`
    CreatedAt       time.Time           `json:"createdAt"`
    UpdatedAt       time.Time           `json:"updatedAt"`
}
```

### 3. Рефакторинг воркеров

#### 3.1 Единый email-воркер

**NATS subject:** `email-notification` (заменяет `verify-email` и `forget-password`)

```go
type EmailNotificationType string

const (
    EmailTypeVerifyEmail    EmailNotificationType = "verify_email"
    EmailTypeForgetPassword EmailNotificationType = "forget_password"
    EmailTypeNewMessage     EmailNotificationType = "new_message"
)

type EmailNotificationDTO struct {
    Type       EmailNotificationType `json:"type"`
    ReceiverID uint64                `json:"receiverId"`
    Payload    json.RawMessage       `json:"payload"`
}
```

Воркер:
1. Десериализует `EmailNotificationDTO` из NATS-сообщения
2. По `Type` десериализует `Payload` в конкретную структуру
3. Для `new_message` — проверяет `EmailConsents` через `SettingsUseCases`. Если бит не установлен — пропускает
4. Системные (`verify_email`, `forget_password`) — отправляются всегда, без проверки согласий
5. Вызывает соответствующий contentbuilder для формирования письма

Type-specific payloads:

```go
// verify_email и forget_password — достаточно ReceiverID из обёртки, payload пустой

// new_message
type NewMessageEmailPayload struct {
    MessageID uint64 `json:"messageId"`
    ChatID    uint64 `json:"chatId"`
}
```

#### 3.2 Единый web-push-воркер

**NATS subject:** `web-push-notification` (остаётся, DTO меняется)

```go
type WebPushNotificationType string

const (
    WebPushTypeNewMessage WebPushNotificationType = "new_message"
)

type WebPushNotificationDTO struct {
    Type       WebPushNotificationType `json:"type"`
    ReceiverID uint64                  `json:"receiverId"`
    Payload    json.RawMessage         `json:"payload"`
}
```

Воркер:
1. Десериализует обёртку
2. Проверяет `WebPushConsents` пользователя — если бит не установлен, пропускает
3. Получает подписки пользователя
4. Отправляет push по каждой подписке

### 4. NATS Subjects

| Было | Стало |
|------|-------|
| `verify-email` | `email-notification` |
| `forget-password` | `email-notification` |
| `web-push-notification` | `web-push-notification` |

Конфиг `NATSSubjects`:
```go
type NATSSubjects struct {
    EmailNotification   string
    WebPushNotification string
}
```

`NATSWorkers`:
```go
type NATSWorkers struct {
    EmailNotification   NATSWorker
    WebPushNotification NATSWorker
}
```

### 5. Публикация в NATS

Места, которые сейчас публикуют в NATS:
- `services/auth/service.go` — `verify-email`, `forget-password` subjects
- `controllers/http/handlers/api/ws/ws.go` — `web-push-notification` subject

Все переходят на новый формат `EmailNotificationDTO` / `WebPushNotificationDTO` с `Type` и `Payload`.

### 6. API

Существующий `UpdateSettings` (`PATCH /api/settings`) расширяется — принимает `emailConsents` и `webPushConsents`. Новых эндпоинтов не нужно.

### 7. UI

В модалке профиля, между «Сменить пароль» и «Тёмная тема» — раздвигающаяся секция:

```
▸ Уведомления
  ┌──────────────────────────────────────┐
  │ Новые сообщения                      │
  │   Email                        [○ ]  │
  │   Web Push                     [○ ]  │
  └──────────────────────────────────────┘
```

Группировка по типу уведомления, внутри — тоглы по каналам.

Тоглы визуально идентичны переключателю темы (`.theme-switch`).

#### Web Push тогл

- **Включение:** запрос разрешения браузера → `PushManager.subscribe` → сохранение подписки через `POST /api/web-push/subscribe` → установка бита `ConsentNewMessage` в `webPushConsents` через `PATCH /api/settings`
- **Выключение:** `subscription.unsubscribe()` → удаление подписки через `DELETE /api/web-push/subscribe/:id` → сброс бита
- **Браузер не поддерживает:** тогл disabled, подсказка «Не поддерживается браузером»

#### Email тогл

- Просто сохраняет/снимает бит `ConsentNewMessage` в `emailConsents` через `PATCH /api/settings`

#### Инициализация состояния тоглов

При открытии профиля:
- Загрузить `Settings` через `GET /api/settings`
- Web Push тогл: `ON` если `webPushConsents & ConsentNewMessage != 0` И есть активная браузерная подписка
- Email тогл: `ON` если `emailConsents & ConsentNewMessage != 0`

### 8. Что НЕ входит в скоуп

- Уведомления о чём-либо кроме новых сообщений
- In-app уведомления (колокольчик)
- Настройка тишины (mute) по чатам
- Email contentbuilder для new_message реализуется в рамках этой задачи (простое текстовое уведомление)
