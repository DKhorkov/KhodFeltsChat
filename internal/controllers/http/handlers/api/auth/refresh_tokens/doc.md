# Пакет auth/refresh_tokens

## Назначение

Обновление access и refresh токенов по refresh токену из cookies.

## Маршрут

`PUT /api/sessions` → `Handler(u, cookiesConfig)`

## Ответы

- `204` — успех, новые токены в cookies
- `401` — невалидный refresh токен
