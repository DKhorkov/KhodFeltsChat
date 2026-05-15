# Пакет usecases/push_subscriptions

## Назначение

Бизнес-логика для работы с push-подписками: создание, получение, удаление и отправка push-уведомлений через Web Push API.

## Ключевые операции

### CreatePushSubscription
- Делегирует вызов в `PushSubscriptionsService`.

### GetPushSubscriptionsByUserID
- Делегирует вызов в `PushSubscriptionsService`.

### DeletePushSubscription
- Делегирует вызов в `PushSubscriptionsService`.

### SendPushNotification
- Формирует JSON-payload с полями `title`, `body`, `chatId` из данных сообщения.
- Отправляет push-уведомление через `webpush-go` с VAPID-ключами из конфигурации.
- При получении HTTP 410 (Gone) или 404 (Not Found) автоматически удаляет невалидную подписку.

## Зависимости

- `internal/interfaces` — `PushSubscriptionsService`.
- `internal/domains` — `PushSubscription`, `Message`.
- `internal/config` — `WebPushConfig`.
- `github.com/SherClockHolmes/webpush-go` — отправка Web Push уведомлений.
- `github.com/DKhorkov/libs/logging` — логирование.
