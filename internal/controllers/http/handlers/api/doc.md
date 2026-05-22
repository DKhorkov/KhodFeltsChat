# Пакет controllers/http/handlers/api

## Назначение

HTTP-обработчики REST API, сгруппированные по предметным областям. Каждый
обработчик читает запрос, вызывает usecase и записывает JSON-ответ.
Все маршруты доступны под префиксом `/api`.

## Auth

| Обработчик          | Метод  | Путь                                  |
|---------------------|--------|---------------------------------------|
| register            | POST   | /api/users                            |
| login               | POST   | /api/sessions                         |
| logout              | DELETE | /api/sessions                         |
| logout_all          | DELETE | /api/sessions/all                     |
| refresh             | PUT    | /api/sessions                         |
| verify_email        | GET    | /api/users/email/verify/{token}       |
| forget_password     | POST   | /api/users/password/forget/{token}    |
| change_password     | POST   | /api/users/password/change            |
| send_verify_email   | POST   | /api/users/email/verify               |
| send_forget_password| POST   | /api/users/password/forget            |

## Users

| Обработчик  | Метод | Путь             |
|-------------|-------|------------------|
| me          | GET   | /api/users/me    |
| update      | PUT   | /api/users/me    |
| user_by_id  | GET   | /api/users/{id}  |
| users       | GET   | /api/users       |

## Settings

| Обработчик      | Метод | Путь                      |
|-----------------|-------|---------------------------|
| get_settings    | GET   | /api/users/me/settings    |
| update_settings | PUT   | /api/users/me/settings    |

## Chats

| Обработчик  | Метод | Путь        |
|-------------|-------|-------------|
| create      | POST  | /api/chats  |
| user_chats  | GET   | /api/chats  |

## Messages

| Обработчик    | Метод | Путь                       |
|---------------|-------|----------------------------|
| chat_messages | GET   | /api/chats/{id}/messages   |

## WebSocket

| Обработчик | Метод | Путь     |
|------------|-------|----------|
| ws         | GET   | /api/ws  |

- Хранит активные соединения в `sync.Map` (userID → conn).
- Аутентифицирует пользователя, обновляет соединение до WebSocket.
- Читает входящие JSON-сообщения в цикле.
- Рассылает сообщения всем онлайн-участникам чата.
- Публикует `WebPushNotificationDTO` в NATS для офлайн-участников чата.

## Web Push (Web Push Notifications)

| Обработчик  | Метод  | Путь                                  |
|-------------|--------|---------------------------------------|
| subscribe   | POST   | /api/web_push/subscribe               |
| unsubscribe | DELETE | /api/web_push/subscribe/{id}          |
| vapid_key   | GET    | /api/web_push/vapid-key               |

## Зависимости

- `internal/usecases/*` — бизнес-логика.
- `internal/controllers/http/schemas` — структуры запросов/ответов.
- `internal/controllers/http/mappers` — конвертация домен → схема.
- `internal/controllers/http/handlers/common` — shared утилиты.
- `gorilla/websocket`.
