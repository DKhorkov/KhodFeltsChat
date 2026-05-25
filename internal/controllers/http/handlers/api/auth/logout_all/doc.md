# Пакет auth/logout_all

## Назначение

Выход из всех сессий. Инвалидирует все refresh токены пользователя и очищает cookies.

## Маршрут

`DELETE /api/sessions/all` → `Handler(u, cookiesConfig)`

## Ответы

- `204` — успех
- `401` — не авторизован
