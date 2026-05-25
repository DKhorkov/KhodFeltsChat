# Пакет handlers/common

## Назначение

Shared-утилиты, используемые всеми HTTP-обработчиками (API и Web).

## Файлы

| Файл | Описание |
|------|----------|
| `headers.go` | Константы HTTP-заголовков: `Content-Type` (JSON, HTML, JS), `Service-Worker-Allowed` |
| `pagination.go` | `GetPaginationFromRequest(r)` — извлекает `limit` и `offset` из query-параметров, возвращает `*domains.Pagination` |
| `route.go` | Константа `IDRouteKey` — ключ для извлечения `{id}` из URL через `mux.Vars` |
