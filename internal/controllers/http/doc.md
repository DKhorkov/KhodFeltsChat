# Пакет controllers/http

## Назначение

HTTP-контроллер на основе `gorilla/mux`. Настраивает маршрутизатор, цепочку
middleware, регистрирует обработчики и управляет жизненным циклом сервера.

## Middleware (применяются глобально, в порядке обёртки)

1. **Tracing** — инжектирует OpenTelemetry span в контекст запроса.
2. **Metrics** — считает HTTP-метрики (Prometheus).
3. **Request-ID** — генерирует и проставляет уникальный ID запроса.
4. **Logging** — логирует метод, путь, статус и длительность.
5. **Cookie JWT Auth** — извлекает access-токен из cookie и валидирует его.

## Bypass аутентификации

Следующие маршруты доступны без JWT-токена:
- `/docs`, `/swagger` — документация.
- `POST /api/sessions` — вход (логин).
- `POST /api/users` — регистрация.
- `GET /api/users` и `GET /api/users/{id}` — публичные данные пользователей.
- `GET /api/users/email/verify/{token}` — подтверждение email.
- `POST /api/users/password/forget/{token}`, `POST /api/users/email/verify`,
  `POST /api/users/password/forget` — сброс/восстановление пароля.

## Graceful Shutdown

Сервер запускается через `http.Server`. При получении сигнала завершения
вызывается `Shutdown` с контекстом-таймаутом для корректного дозавершения
активных соединений.

## Зависимости

- `gorilla/mux` — маршрутизация.
- `internal/controllers/http/handlers` — регистрация обработчиков.
- `internal/interfaces` — usecase-интерфейсы.
- OpenTelemetry, Prometheus.
