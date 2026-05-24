# Пакет handlers/web/service_worker

## Назначение

Отдаёт Service Worker (`sw.js`) с правильными заголовками.

## Маршрут

`GET /web/sw.js` → `Handler()`

## Реализация

Устанавливает `Content-Type: application/javascript` и `Service-Worker-Allowed: /`, затем отдаёт файл `static/sw.js` через `http.ServeFile`.
