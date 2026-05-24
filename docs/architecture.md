# Архитектура KhodFeltsChat

## Обзор

Чат-приложение на Go с real-time WebSocket, REST API, PostgreSQL, Redis, NATS и OpenTelemetry трассировкой.

## Clean Architecture

```
┌─────────────────────────────────────────────┐
│              Controllers / Handlers          │  ← HTTP, WebSocket
├─────────────────────────────────────────────┤
│                  Use Cases                   │  ← Orchestration, валидация
├─────────────────────────────────────────────┤
│                  Services                    │  ← Бизнес-логика, UoW транзакции
├─────────────────────────────────────────────┤
│                Repositories                  │  ← Доступ к данным (PG, SMTP)
├─────────────────────────────────────────────┤
│                  Domains                     │  ← Модели, DTO
└─────────────────────────────────────────────┘
```

Каждый слой обёрнут **trace decorator** для OpenTelemetry. Dependency injection — через конструкторы в `cmd/main.go`.

## Доменные модели

| Модель | Описание |
|--------|----------|
| User | id, username, email, emailConfirmed, password, timestamps |
| Chat | id, title, description, type (private/group), members, messages, isRead |
| Message | id, chatID, sender, text, isRead, timestamps (builder pattern) |
| RefreshToken | id, userID, TTL, value, timestamps |
| Pagination | optional Limit/Offset |

### Валидация Chat
- `private` — ровно 2 участника
- Любой тип — минимум 1 участник

## Инфраструктура

### PostgreSQL
- Основное хранилище данных
- Query builder: `squirrel`
- Миграции: `goose`
- Таблицы: `users`, `refresh_tokens`, `chats`, `chats_members`, `messages`, `messages_statuses`

### Redis
- Rate limiting email операций: 3 попытки за 3 минуты
- Одноразовые токены (verify email, forget password): TTL 15 минут
- Гарантия: один токен на пользователя (новый инвалидирует старый)
- При использовании токен удаляется из кэша

### NATS
- Асинхронная отправка уведомлений
- Subjects: `verify-email`, `forget-password`
- Payload: `{UserID: int}`
- Workers (consumers) вызывают notifications usecases

### OpenTelemetry + Jaeger
- Трассировка на всех слоях: repositories, services, usecases, handlers
- Jaeger UI: `:16686`

### Prometheus + Grafana
- Метрики middleware в HTTP
- Prometheus: `:9090`, Grafana: `:3000`

## HTTP API

### Аутентификация
| Метод | Путь | Описание |
|-------|------|----------|
| POST | /users | Регистрация |
| POST | /sessions | Логин |
| PUT | /sessions | Обновление токенов |
| DELETE | /sessions | Логаут |
| GET | /users/email/verify/{token} | Подтверждение email |
| POST | /users/password/forget/{token} | Сброс пароля |
| POST | /users/password/change | Смена пароля |
| POST | /users/email/verify | Отправка письма подтверждения |
| POST | /users/password/forget | Отправка письма сброса пароля |

### Пользователи
| Метод | Путь | Описание |
|-------|------|----------|
| GET | /users | Список (фильтр по username, пагинация) |
| GET | /users/{id} | Профиль по ID |
| GET | /users/me | Текущий пользователь |
| PUT | /users/me | Обновление профиля |

### Чаты и сообщения
| Метод | Путь | Описание |
|-------|------|----------|
| POST | /chats | Создание чата |
| GET | /chats | Чаты текущего пользователя |
| GET | /chats/{id}/messages | Сообщения чата (с пагинацией) |

### WebSocket
- Upgrade из HTTP с cookie-аутентификацией
- JSON формат: `{"chatId": N, "text": "..."}`
- Мультисессия: один пользователь может иметь несколько WebSocket-соединений (разные устройства/вкладки)
- `sync.Map[userID → *userConnections]`, где `userConnections` содержит `sync.Mutex` + `[]*websocket.Conn`
- Fan-out: события (`new_message`, `message_deleted`) доставляются **всем** соединениям **всех** участников чата, включая отправителя
- Клиенты не обновляют UI самостоятельно при отправке/удалении — только по WS-событиям от сервера (single source of truth)

## Middleware (порядок)

1. Tracing (OpenTelemetry span)
2. Metrics (Prometheus)
3. Request ID
4. Logging
5. Cookie-based JWT auth (с bypass для публичных эндпоинтов)

## Безопасность

- **JWT**: access token + refresh token в HTTP-only cookies
- **bcrypt**: хэширование паролей
- **Base64 токены**: одноразовые для email verify и password reset
- **Salt**: UUID salt + userID, закодированные в base64
- **Rate limiting**: Redis-backed, 3 попытки / 3 минуты на email операции
- **Ротация токенов**: при логине старый refresh token удаляется

## Unit of Work

```go
uow.Do(ctx, func(ctx context.Context, tx pg.Transaction) error {
    // Все операции в одной транзакции
    // Rollback при ошибке или context cancellation
    // Commit при успехе
})
```

Repositories создаются factory-функциями, принимающими `pg.Transaction`.

## Деплой

- **Local**: `task local` (docker infra) + `go run cmd/main.go`
- **Production**: `task prod` (full docker-compose stack)
- **CI**: GitHub Actions — linting (`golangci-lint`) + тесты
