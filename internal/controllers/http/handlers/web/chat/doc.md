# Пакет handlers/web/chat

## Назначение

Отдаёт HTML-страницу чатов.

## Маршрут

`GET /web/chat` → `Handler()`

## Реализация

Шаблон `chat.html` + `navbar.html` парсится один раз (`sync.Once`). При ошибке — `RenderError` с кодом 500.
