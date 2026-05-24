# Пакет handlers/web/login

## Назначение

Отдаёт HTML-страницу входа и регистрации.

## Маршрут

`GET /web/login` → `Handler()`

## Реализация

Шаблон `login.html` + `navbar.html` парсится один раз (`sync.Once`). При ошибке — `RenderError` с кодом 500.
