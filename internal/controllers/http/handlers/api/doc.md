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

| Обработчик    | Метод  | Путь                      |
|---------------|--------|---------------------------|
| me            | GET    | /api/users/me             |
| update        | PUT    | /api/users/me             |
| update_avatar | PUT    | /api/users/me/avatar      |
| delete_avatar | DELETE | /api/users/me/avatar      |
| user_by_id    | GET    | /api/users/{id}           |
| users         | GET    | /api/users                |

### update_avatar
Загружает или заменяет аватар текущего пользователя. Принимает `multipart/form-data` с полем `avatar`. Сохраняет файл через `FileStorageService` и обновляет `avatar_path` пользователя. Возвращает `AvatarURL` — публично доступный URL для скачивания аватара.

### delete_avatar
Удаляет аватар текущего пользователя. Вызывает `UsersUseCases.DeleteAvatar`, который удаляет файл из хранилища и сбрасывает `avatar_path`.

## Files

| Обработчик      | Метод | Путь                   |
|-----------------|-------|------------------------|
| download_file   | GET   | /api/files/{filename}  |

### download_file
Отдаёт содержимое файла из локального хранилища по имени файла. Маршрут не требует аутентификации (входит в bypass-список). Используется для публичного доступа к аватарам пользователей.

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

| Обработчик    | Метод  | Путь                       |
|---------------|--------|----------------------------|
| chat_messages | GET    | /api/chats/{id}/messages   |
| delete        | DELETE | /api/messages/{id}         |
| update        | PUT    | /api/messages/{id}         |

### delete
Удаляет сообщение. Body JSON: `{"forAll": bool}`. Всегда получает сообщение через `GetMessageByID` для определения `chatID`. Если `forAll=true` — удаляет для всех, рассылает WS-событие `message_deleted` всем участникам чата через `BroadcastMessageDeleted`. Если `forAll=false` — удаляет только для текущего пользователя, отправляет WS-событие `message_deleted` только на его соединения через `SendMessageDeletedToUser`.

### update
Редактирует текст сообщения. Body JSON: `{"text": string}`. Только автор может редактировать. Рассылает WS-событие `message_edited` всем участникам чата через `BroadcastMessageEdited`.

## WebSocket

| Обработчик | Метод | Путь     |
|------------|-------|----------|
| ws         | GET   | /api/ws  |

- Хранит активные соединения в `sync.Map` (userID → `*userConnections`), поддерживает мультисессию (несколько соединений на пользователя).
- Аутентифицирует пользователя, обновляет соединение до WebSocket.
- Читает входящие JSON-сообщения в цикле.
- Оборачивает исходящие сообщения в `WSEvent` envelope (`type` + `payload`).
- Рассылает `new_message` событие всем онлайн-участникам чата.
- Реализует `WSBroadcaster` — метод `BroadcastMessageDeleted` рассылает `message_deleted` событие всем участникам чата (удаление у всех); `SendMessageDeletedToUser` отправляет только конкретному пользователю (удаление у себя); `BroadcastMessageEdited` рассылает `message_edited` событие всем участникам чата.
- Публикует `WebPushNotificationDTO` и `EmailNotificationDTO` в NATS для офлайн-участников чата.

## Web Push (Web Push Notifications)

| Обработчик  | Метод  | Путь                                  |
|-------------|--------|---------------------------------------|
| subscribe   | POST   | /api/web_push/subscribe               |
| unsubscribe | DELETE | /api/web_push/subscribe/{id}          |
| vapid_key   | GET    | /api/web_push/vapid-key               |

## Зависимости

- `internal/usecases/*` — бизнес-логика.
- `internal/interfaces` — включая `FileStorageUseCases` для обработчиков файлов/аватаров.
- `internal/controllers/http/schemas` — структуры запросов/ответов; содержит `AvatarURL` и `FileDownloadURL`.
- `internal/controllers/http/mappers` — конвертация домен → схема.
- `internal/controllers/http/handlers/common` — shared утилиты.
- `gorilla/websocket`.
