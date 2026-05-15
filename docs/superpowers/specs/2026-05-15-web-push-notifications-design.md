# Web Push Notifications — Design

## Цель

Отправлять нативные push-уведомления о новых сообщениях пользователям, у которых нет активного WebSocket соединения (вкладка закрыта / браузер свёрнут). Онлайн-пользователи продолжают получать toast-уведомления через существующий WS.

## Общий поток данных

```
Новое сообщение через WebSocket
  -> WS handler сохраняет в БД, рассылает онлайн-участникам
  -> Для каждого офлайн-участника: publish в NATS subject "push-notification"
     payload: { userID, messageID }
  -> push-notification worker получает событие
  -> Достаёт сообщение из БД (senderName, text, chatID)
  -> Достаёт push-подписки пользователя из БД
  -> Отправляет Web Push через webpush-go (VAPID)
  -> Если пуш-сервер вернул 410/404 -> удаляет подписку из БД
```

WS handler публикует **по одному NATS-событию на каждого офлайн-участника**. Worker остаётся простым: одно событие = один пользователь.

## Схема БД

Миграция `20260515000000_push_subscriptions.sql`:

```sql
CREATE TABLE push_subscriptions (
    id         BIGSERIAL    PRIMARY KEY,
    user_id    BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint   TEXT         NOT NULL UNIQUE,
    p256dh     TEXT         NOT NULL,
    auth       TEXT         NOT NULL,
    created_at TIMESTAMP    NOT NULL DEFAULT now()
);

CREATE INDEX idx_push_subscriptions_user_id ON push_subscriptions(user_id);
```

- `UNIQUE` на `endpoint` — страховка от дубликатов на уровне БД.
- У одного пользователя может быть несколько подписок (разные браузеры/устройства).

## API (тег: web-pushes)

### `GET /api/push/vapid-key`

Возвращает публичный VAPID ключ для фронтенда.

**Response:** `200 OK`
```json
{
  "publicKey": "BEl62i..."
}
```

### `POST /api/push/subscribe`

Создаёт push-подписку для текущего пользователя.

**Request:**
```json
{
  "endpoint": "https://fcm.googleapis.com/...",
  "keys": {
    "p256dh": "BNcRd...",
    "auth": "tBHI..."
  }
}
```

**Response:** `201 Created`
```json
{
  "id": 42
}
```

### `DELETE /api/push/subscribe/{id}`

Удаляет push-подписку по ID.

**Response:** `204 No Content`

## Домен

Добавляется в `internal/domains/notifications.go`:

```go
type PushNotificationDTO struct {
    UserID    uint64 `json:"userId"`
    MessageID uint64 `json:"messageId"`
}
```

Новая модель `internal/domains/push_subscription.go`:

```go
type PushSubscription struct {
    ID        uint64
    UserID    uint64
    Endpoint  string
    P256dh    string
    Auth      string
    CreatedAt time.Time
}
```

## Интерфейсы

### Repository

```go
type PushSubscriptionsRepository interface {
    CreatePushSubscription(ctx context.Context, subscription domains.PushSubscription) (uint64, error)
    GetPushSubscriptionsByUserID(ctx context.Context, userID uint64) ([]domains.PushSubscription, error)
    DeletePushSubscription(ctx context.Context, id uint64) error
}
```

### Service

```go
type PushSubscriptionsService interface {
    PushSubscriptionsRepository
}
```

### UseCases

```go
type PushSubscriptionsUseCases interface {
    CreatePushSubscription(ctx context.Context, subscription domains.PushSubscription) (*domains.PushSubscription, error)
    GetPushSubscriptionsByUserID(ctx context.Context, userID uint64) ([]domains.PushSubscription, error)
    DeletePushSubscription(ctx context.Context, id uint64) error
    SendPushNotification(ctx context.Context, subscription domains.PushSubscription, message domains.Message) error
}
```

## NATS

### Новый subject: `push-notification`

Конфигурация добавляется в `NATSSubjects`, `NATSWorkers`.

### Worker: `push-notification-worker`

Файл: `internal/workers/handlers/builders/push_notification/builder.go`

Структура аналогична `verify_email/builder.go`:

1. Десериализовать NATS-сообщение в `PushNotificationDTO`
2. Получить сообщение из БД по `messageID` (senderName, text, chatID)
3. Получить все push-подписки пользователя по `userID`
4. Для каждой подписки — отправить Web Push через `webpush-go`
5. Если пуш-сервер вернул 410 Gone / 404 — удалить подписку из БД
6. Остальные ошибки — логировать

## VAPID конфигурация

Генерация ключей (одноразово):

```bash
go run github.com/SherClockHolmes/webpush-go/cmd/vapid-keygen
```

Добавляется в `Config`:

```go
type WebPushConfig struct {
    VAPIDPublicKey  string  // env: VAPID_PUBLIC_KEY
    VAPIDPrivateKey string  // env: VAPID_PRIVATE_KEY
    VAPIDContact    string  // env: VAPID_CONTACT, например "mailto:admin@kfc.com"
}
```

## WS handler: изменения

В цикле по `chatMembers` в `ws.go`, когда `h.connections.Load(member.ID)` возвращает `false`:

```go
// Участник офлайн — публикуем событие для push-notification worker:
pushDTO := domains.PushNotificationDTO{
    UserID:    member.ID,
    MessageID: savedMessage.ID,
}
// publish to NATS "push-notification" subject
```

Для этого WS handler получает зависимость на NATS publisher.

## Фронтенд

### Service Worker: `sw.js`

Новый файл в корне статики. Слушает push-события и показывает нативное уведомление:

```js
self.addEventListener('push', (event) => {
    const data = event.data.json();
    event.waitUntil(
        self.registration.showNotification(data.title, {
            body: data.body,
            data: { chatId: data.chatId }
        })
    );
});

self.addEventListener('notificationclick', (event) => {
    event.notification.close();
    event.waitUntil(clients.openWindow('/web/home'));
});
```

### Инициализация в `chat.js`

В `DOMContentLoaded` после `connectWebSocket()` вызывается `initPushNotifications()`:

1. Проверить поддержку Push API
2. Зарегистрировать Service Worker
3. `registration.pushManager.getSubscription()`
4. Если подписка есть — отправить на бэкенд (на случай если потеряна в БД)
5. Если подписки нет:
   - `Notification.permission === "granted"` — подписаться автоматически
   - `"default"` — запросить разрешение, при согласии подписаться
   - `"denied"` — ничего не делать
6. Сохранить subscription ID из ответа бэкенда в `localStorage`

### Кнопка в Settings

"Включить уведомления" / "Отключить уведомления":

- **Включить** — запускает флоу подписки (шаги 3-5 выше)
- **Отключить** — вызывает `subscription.unsubscribe()` + `DELETE /api/push/subscribe/{id}` + очищает `localStorage`

## Зависимости

Новая Go-зависимость:

```
github.com/SherClockHolmes/webpush-go
```

## Tracing

Добавляются span-конфиги для нового слоя:

- `Repositories.PushSubscriptions`
- `Services.PushSubscriptions`
- `UseCases.PushSubscriptions`
- `Handlers.PushNotification`
