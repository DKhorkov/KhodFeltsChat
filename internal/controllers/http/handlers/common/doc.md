# Пакет handlers/common

## Назначение

Shared-утилиты, используемые всеми HTTP-обработчиками (API и Web).

## Файлы

| Файл | Описание |
|------|----------|
| `headers.go` | Константы HTTP-заголовков: `Content-Type` (JSON, HTML, JS, text/plain, image/jpeg), `Service-Worker-Allowed` |
| `pagination.go` | `GetPaginationFromRequest(r)` — извлекает `limit` и `offset` из query-параметров, возвращает `*domains.Pagination` |
| `route.go` | Константы `IDRouteKey` (для `{id}`) и `ReactionIDRouteKey` (для `{reactionId}` в DELETE /messages/{id}/reactions/{reactionId}) — ключи для `mux.Vars` |
