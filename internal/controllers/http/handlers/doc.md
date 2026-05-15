# Пакет controllers/http/handlers

## Назначение

Оркестратор HTTP-маршрутизации. Регистрирует общие обработчики (404, 405, docs,
metrics) и делегирует API-маршруты в подпакет `api/` через subrouter с префиксом
`/api`.

## Структура

```
handlers/
├── setup.go        — оркестратор: subrouter /api, /web, docs, metrics, default, not_allowed
├── common/         — shared утилиты (pagination, route keys, headers)
├── default/        — 404 handler (перенаправляет на /docs)
├── not_allowed/    — 405 handler
├── docs/           — Swagger UI (статические файлы)
├── api/            — API-обработчики (см. api/doc.md)
└── web/            — веб-интерфейс: страницы, Service Worker (/sw.js), статика (CSS/JS)
```

## Shared-пакеты

- **common** — константы роутинга (`IDRouteKey`), пагинация, хелперы заголовков.
- **default** — перенаправляет на `/docs`.
- **not_allowed** — возвращает 405.
- **docs** — отдаёт Swagger UI.

## Зависимости

- `internal/controllers/http/handlers/api` — регистрация API-обработчиков.
- `gorilla/mux` — маршрутизация.
- `go-openapi/runtime/middleware` — Swagger UI.
- `prometheus/client_golang` — метрики.
