# Пакет handlers/web/forget_password

## Назначение

Отдаёт HTML-страницу сброса пароля.

## Маршрут

`GET /web/forget-password` → `Handler()`

## Реализация

Шаблон `forget_password.html` + `navbar.html` парсится один раз (`sync.Once`). Передаёт `email` из query-параметра в шаблон. При ошибке — `RenderError` с кодом 500.
