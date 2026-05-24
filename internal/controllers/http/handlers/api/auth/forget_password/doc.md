# Пакет auth/forget_password

## Назначение

Сброс пароля по токену из письма.

## Маршрут

`POST /api/users/password/forget/{token}` → `Handler(u)`

## Body

`{"newPassword": "..."}`

## Ответы

- `204` — пароль сброшен
- `400` — некорректный запрос
- `404` — токен не найден
