# Пакет handlers/not_allowed

## Назначение

Обработчик для запросов с некорректным HTTP-методом. Возвращает `405 Method Not Allowed` с описанием метода и URL.

## Маршрут

Назначается через `mux.Router.MethodNotAllowedHandler`.
