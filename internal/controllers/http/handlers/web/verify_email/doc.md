# Пакет handlers/web/verify_email

## Назначение

Отдаёт HTML-страницу подтверждения email.

## Маршрут

`GET /web/verify-email/{token}` → `Handler()`

## Реализация

Шаблон `verify_email.html` парсится один раз (`sync.Once`). Константа `TokenRouteKey` используется для извлечения токена из URL. При ошибке — `RenderError` с кодом 500.
