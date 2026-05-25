# Пакет auth/logout

## Назначение

Выход из текущей сессии. Инвалидирует refresh токен и очищает cookies.

## Маршрут

`DELETE /api/sessions` → `Handler(u, cookiesConfig)`

## Ответы

- `204` — успех
- `401` — не авторизован
