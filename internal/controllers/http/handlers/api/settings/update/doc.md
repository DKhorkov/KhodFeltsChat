# Пакет settings/update

## Назначение

Обновление настроек текущего пользователя.

## Маршрут

`PUT /api/users/me/settings` → `Handler(s)`

## Body

`{"theme": "...", "emailConsents": N, "webPushConsents": N}`

## Ответы

- `204` — настройки обновлены
- `400` — некорректный запрос
- `401` — не авторизован
