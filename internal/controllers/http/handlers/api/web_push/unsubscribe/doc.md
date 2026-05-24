# Пакет web_push/unsubscribe

## Назначение

Удаление web-push подписки по ID.

## Маршрут

`DELETE /api/web-push/subscriptions/{id}` → `Handler(u)`

## Ответы

- `204` — подписка удалена
- `400` — некорректный ID
- `401` — не авторизован
- `404` — подписка не найдена
