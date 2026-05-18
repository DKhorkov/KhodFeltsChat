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
6. При получении любого другого 4xx/5xx возвращает ошибку
   `ErrWebPushNotificationRejected` с телом ответа для диагностики.

## Конфигурация

Используется `config.WebPushConfig`:

| Поле | Env-переменная | Описание | По умолчанию |
|------|---------------|----------|--------------|
| `VAPIDPublicKey` | `VAPID_PUBLIC_KEY` | Публичный VAPID-ключ | — |
| `VAPIDPrivateKey` | `VAPID_PRIVATE_KEY` | Приватный VAPID-ключ | — |
| `VAPIDContact` | `VAPID_CONTACT` | Email для VAPID subject (**без** `mailto:` — библиотека добавляет его сама) | `admin@example.com` |
| `TTL` | `WEB_PUSH_TTL` | Время жизни push-уведомления (сек) | `86400` |

### VAPID_CONTACT — важный нюанс

Библиотека `webpush-go` автоматически добавляет `mailto:` к subscriber в JWT
(см. `vapid.go:76-78`). Если указать `mailto:user@example.com`, в JWT попадёт
`mailto:mailto:user@example.com` — Chrome это принимает, но Apple APNs
отвечает `403 BadJwtToken`. Поэтому значение **не должно** содержать `mailto:`.

Генератор ключей: `scripts/vapidgen.go`.

## Зависимости

- `github.com/SherClockHolmes/webpush-go` — отправка push-уведомлений по Web Push Protocol
- `github.com/DKhorkov/kfc/internal/config.WebPushConfig` — VAPID ключи, TTL, email для subject
- `github.com/DKhorkov/libs/logging.Logger` — логирование ошибок
- `github.com/DKhorkov/kfc/internal/domains` — типы `WebPushSubscription`, `Message`
- `github.com/DKhorkov/kfc/internal/errors` — `ErrWebPushSubscriptionExpired`

## Trace-декоратор

`TraceDecorator` оборачивает все методы репозитория трассировкой OpenTelemetry.
