# Пакет controllers/http/handlers

## Назначение

HTTP-обработчики, сгруппированные по предметным областям. Каждый обработчик
читает запрос, вызывает usecase и записывает JSON-ответ.

## Auth (`/auth`)

| Обработчик          | Метод  | Путь                              |
|---------------------|--------|-----------------------------------|
| register            | POST   | /users                            |
| login               | POST   | /sessions                         |
| logout              | DELETE | /sessions                         |
| refresh             | PUT    | /sessions                         |
| verify_email        | GET    | /users/email/verify/{token}       |
| forget_password     | POST   | /users/password/forget/{token}    |
| change_password     | POST   | /users/password/change            |
| send_verify_email   | POST   | /users/email/verify               |
| send_forget_password| POST   | /users/password/forget            |

## Users (`/users`)

| Обработчик  | Метод | Путь         |
|-------------|-------|--------------|
| me          | GET   | /users/me    |
| update      | PUT   | /users/me    |
| user_by_id  | GET   | /users/{id}  |
| users       | GET   | /users       |

## Chats (`/chats`)

| Обработчик  | Метод | Путь    |
|-------------|-------|---------|
| create      | POST  | /chats  |
| user_chats  | GET   | /chats  |

## Messages

| Обработчик    | Метод | Путь                   |
|---------------|-------|------------------------|
| chat_messages | GET   | /chats/{id}/messages   |

## WebSocket (`/ws`)

- Хранит активные соединения в `sync.Map` (userID → conn).
- Аутентифицирует пользователя, обновляет соединение до WebSocket.
- Читает входящие JSON-сообщения в цикле.
- Рассылает сообщения всем онлайн-участникам чата.

## Прочие

- **Default** — перенаправляет на `/docs`.
- **Docs** — отдаёт Swagger UI (статические файлы).

## Зависимости

- `internal/usecases/*` — бизнес-логика.
- `internal/controllers/http/schemas` — структуры запросов/ответов.
- `internal/controllers/http/mappers` — конвертация домен → схема.
- `gorilla/websocket`.
