# Пакет usecases/web_push_subscriptions

## Назначение

Бизнес-логика для работы с push-подписками: создание, получение, удаление и отправка push-уведомлений через Web Push API.

## Ключевые операции

### CreateWebPushSubscription
- Делегирует вызов в `WebPushSubscriptionsService`.

### GetWebPushSubscriptionsByUserID
- Делегирует вызов в `WebPushSubscriptionsService`.

### DeleteWebPushSubscription
- Делегирует вызов в `WebPushSubscriptionsService`.

### SendWebPushNotification
- Формирует JSON-payload с полями `title`, `body`, `chatId` из данных сообщения.
- Отправляет push-уведомление через `webpush-go` с VAPID-ключами из конфигурации.
- При получении HTTP 410 (Gone) или 404 (Not Found) автоматически удаляет невалидную подписку.

## Зависимости

- `internal/interfaces` — `WebPushSubscriptionsService`.
- `internal/domains` — `WebPushSubscription`, `Message`.
- `internal/config` — `WebPushConfig`.
- `github.com/SherClockHolmes/webpush-go` — отправка Web Push уведомлений.
- `github.com/DKhorkov/libs/logging` — логирование.
