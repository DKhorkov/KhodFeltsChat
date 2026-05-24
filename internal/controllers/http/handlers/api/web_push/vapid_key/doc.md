# Пакет web_push/vapid_key

## Назначение

Возвращает VAPID публичный ключ для подписки на web-push уведомления.

## Маршрут

`GET /api/web-push/vapid-key` → `Handler(vapidPublicKey)`

## Ответы

- `200` — JSON с VAPID ключом
