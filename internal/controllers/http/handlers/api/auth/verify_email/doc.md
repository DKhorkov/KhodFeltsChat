# Пакет auth/verify_email

## Назначение

Подтверждение email по токену из письма.

## Маршрут

`GET /api/users/email/verify/{token}` → `Handler(u)`

## Ответы

- `204` — email подтверждён
- `404` — токен не найден
