# repositories/web_push

## Назначение

Репозиторий для отправки Web Push уведомлений через VAPID. Не взаимодействует
с базой данных — доставляет push-уведомления напрямую через Web Push Protocol.

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `SendNotification(ctx, subscription, message)` | Формирует JSON payload и отправляет push-уведомление по подписке |

Метод:
1. Формирует JSON payload с полями `title`, `body`, `chatId`, `timestamp`.
2. Отправляет через `webpush.SendNotification` с VAPID-ключами.
3. Устанавливает `TTL` из конфигурации (по умолчанию 86400 сек / 24 часа) —
   необходимо для iOS, где APNs отбрасывает push без TTL или с TTL=0.
4. Устанавливает `Urgency: High` — пробуждает устройство на iOS.
5. При получении 404/410 (подписка истекла) возвращает ошибку
   `ErrWebPushSubscriptionExpired` для удаления невалидной подписки.

## Конфигурация

Используется `config.WebPushConfig`:

| Поле | Env-переменная | Описание | По умолчанию |
|------|---------------|----------|--------------|
| `VAPIDPublicKey` | `WEB_PUSH_VAPID_PUBLIC_KEY` | Публичный VAPID-ключ | — |
| `VAPIDPrivateKey` | `WEB_PUSH_VAPID_PRIVATE_KEY` | Приватный VAPID-ключ | — |
| `VAPIDContact` | `WEB_PUSH_VAPID_CONTACT` | Email для VAPID subject | — |
| `TTL` | `WEB_PUSH_TTL` | Время жизни push-уведомления (сек) | `86400` |

## Зависимости

- `github.com/SherClockHolmes/webpush-go` — отправка push-уведомлений по Web Push Protocol
- `github.com/DKhorkov/kfc/internal/config.WebPushConfig` — VAPID ключи, TTL, email для subject
- `github.com/DKhorkov/libs/logging.Logger` — логирование ошибок
- `github.com/DKhorkov/kfc/internal/domains` — типы `WebPushSubscription`, `Message`
- `github.com/DKhorkov/kfc/internal/errors` — `ErrWebPushSubscriptionExpired`

## Trace-декоратор

`TraceDecorator` оборачивает все методы репозитория трассировкой OpenTelemetry.
