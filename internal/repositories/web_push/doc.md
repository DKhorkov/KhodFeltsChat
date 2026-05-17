# repositories/web_push

## Назначение

Репозиторий для отправки Web Push уведомлений через VAPID. Не взаимодействует
с базой данных — доставляет push-уведомления напрямую через Web Push Protocol.

## Ключевые методы

| Метод | Описание |
|-------|----------|
| `SendNotification(ctx, subscription, message)` | Формирует JSON payload и отправляет push-уведомление по подписке |

Метод:
1. Формирует JSON payload с полями `title`, `body`, `url`.
2. Отправляет через `webpush.SendNotification` с VAPID-ключами.
3. При получении 404/410 (подписка истекла) возвращает `nil` (graceful degradation).

## Зависимости

- `github.com/SherClockHolmes/webpush-go` — отправка push-уведомлений по Web Push Protocol
- `github.com/DKhorkov/kfc/internal/config.WebPushConfig` — VAPID ключи, email для subject
- `github.com/DKhorkov/libs/logging.Logger` — логирование ошибок
- `github.com/DKhorkov/kfc/internal/domains` — типы `WebPushSubscription`, `Message`

## Trace-декоратор

`TraceDecorator` оборачивает все методы репозитория трассировкой OpenTelemetry.
