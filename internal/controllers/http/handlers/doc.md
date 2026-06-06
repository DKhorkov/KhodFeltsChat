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

## Веб-интерфейс (web/)

### Service Worker (sw.js)

Обрабатывает push-уведомления и клики по ним. На iOS `showNotification`
вызывается **всегда** (iOS отбрасывает push без этого вызова и может
отозвать разрешение). На десктопе уведомление подавляется, если чат в фокусе.

### PWA-манифест (manifest.json)

Поддержка «Добавить на экран Домой». Обязателен для web-push на iOS (16.4+).
Иконки с `purpose: "any"`. Поле `id` предотвращает потерю подписки при
переустановке PWA (iOS 17+).

### Навбар (navbar.js)

- **Тема**: загружается с сервера, применяется через `data-bs-theme`.
  Синхронизируется при возврате на вкладку (`visibilitychange`) и при
  открытии модалки профиля.
- **Web Push подписка**: ключи кодируются в base64url, `applicationServerKey`
  передаётся как `ArrayBuffer` (через `.buffer.slice(0)` — iOS не принимает
  `Uint8Array`).
- **Баннер разрешения (iOS)**: на iOS `Notification.requestPermission()` требует
  user gesture. Если серверный consent включён, но разрешение не запрошено —
  показывается баннер «Разрешите уведомления» вверху страницы. Используется
  `sessionStorage` для показа один раз за сессию PWA. На десктопе баннер не
  показывается — разрешение запрашивается напрямую.

### HTML-шаблоны

Все шаблоны включают Apple meta-теги (`apple-mobile-web-app-capable`,
`apple-mobile-web-app-status-bar-style`), `apple-touch-icon` и ссылку
на `manifest.json`.

### Модалка профиля

Фиксированная высота `80vh` с `overflow-y: auto` — при раскрытии секций
(редактирование, пароль, уведомления) размер модалки не меняется, содержимое
скроллится внутри.

## Shared-пакеты

- **common** — константы роутинга (`IDRouteKey`), пагинация, хелперы заголовков.
- **default** — перенаправляет на `/docs`.
- **not_allowed** — возвращает 405.
- **docs** — отдаёт Swagger UI.

## Зависимости

- `internal/controllers/http/handlers/api` — регистрация API-обработчиков; принимает `fileStorageUseCases` (`interfaces.FileStorageUseCases`) как дополнительный параметр для маршрутов работы с файлами.
- `gorilla/mux` — маршрутизация.
- `go-openapi/runtime/middleware` — Swagger UI.
- `prometheus/client_golang` — метрики.
